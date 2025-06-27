package azureblob

import (
	"encoding/xml"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	uploadablepkg "github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/azureblob/uploadable"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/dustin/go-humanize"
	"github.com/go-chi/render"
	"io"
	"log"
	"net/http"
	"strconv"
)

// As documented in Put Block List's documentation[1]
//
// [1]: https://learn.microsoft.com/en-us/rest/api/storageservices/put-block-list?tabs=microsoft-entra-id#request-body
type blockList struct {
	XMLName     xml.Name `xml:"BlockList"`
	Committed   []string `xml:"Committed"`
	Uncommitted []string `xml:"Uncommitted"`
	Latest      []string `xml:"Latest"`
}

func (azureBlob *AzureBlob) putBlobAbstract(writer http.ResponseWriter, request *http.Request) {
	switch request.URL.Query().Get("comp") {
	case "block":
		azureBlob.putBlock(writer, request)
	case "blocklist":
		azureBlob.putBlockList(writer, request)
	default:
		azureBlob.putBlob(writer, request)
	}
}

func (azureBlob *AzureBlob) putBlob(writer http.ResponseWriter, request *http.Request) {
	key := request.PathValue("key")

	// Parse the Content-Length header
	contentLength, err := strconv.ParseUint(request.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		fail(writer, request, http.StatusBadRequest, "failed to parse the Content-Length header value",
			"key", key, "err", err, "value", contentLength)

		return
	}

	// Generate cache upload URL
	generateCacheUploadURLResponse, err := client.CirrusClient.GenerateCacheUploadURL(
		request.Context(),
		&api.CacheKey{
			TaskIdentification: client.CirrusTaskIdentification,
			CacheKey:           key,
		},
	)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "failed to generate cache upload URL",
			"key", key, "err", err)

		return
	}

	// Proxy cache entry upload since Azure Blob client does not support redirects
	req, err := http.NewRequestWithContext(request.Context(), http.MethodPut,
		generateCacheUploadURLResponse.Url, request.Body)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "failed to create request to proxy "+
			"cache entry upload", "key", key, "err", err)

		return
	}

	// Content-Length is required to avoid HTTP 411
	req.ContentLength = int64(contentLength)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "failed to perform request to proxy "+
			"cache entry upload", "key", key, "err", err)

		return
	}

	if resp.StatusCode != http.StatusOK {
		fail(writer, request, http.StatusInternalServerError, fmt.Sprintf("failed to perform request to proxy "+
			"cache entry upload, got unexpected HTTP %d", resp.StatusCode), "key", key)

		return
	}

	writer.WriteHeader(http.StatusCreated)
}

func (azureBlob *AzureBlob) putBlock(writer http.ResponseWriter, request *http.Request) {
	key := request.PathValue("key")

	cacheKey := &api.CacheKey{
		TaskIdentification: client.CirrusTaskIdentification,
		CacheKey:           key,
	}

	// Decode the block ID
	blockID := request.URL.Query().Get("blockid")

	partNumber, err := blockIDToPartNumber(blockID)
	if err != nil {
		fail(writer, request, http.StatusBadRequest, "failed to extract the block index "+
			"from block ID", "key", key, "blockid", blockID, "err", err)

		return
	}

	// Parse the Content-Length header
	contentLength, err := strconv.ParseUint(request.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		fail(writer, request, http.StatusBadRequest, "failed to parse the Content-Length header value",
			"key", key, "blockid", blockID, "err", err, "value", contentLength)

		return
	}

	// Retrieve an already existing uploadable or compute a new one
	uploadable, _ := azureBlob.uploadables.LoadOrCompute(key, func() *uploadablepkg.Uploadable {
		// Upload locally if we're observing chunks that are smaller
		// than S3's minimum part size for multipart uploads
		local := contentLength < 5*humanize.MiByte

		if local {
			log.Printf("Using local uploadable because the first observed "+
				"block size for key %q is only %s", key, humanize.IBytes(contentLength))
		}

		return uploadablepkg.New(local)
	})

	if uploadable.Local() {
		if err := uploadable.AppendPartLocal(partNumber, request.Body); err != nil {
			fail(writer, request, http.StatusInternalServerError, "failed to upload part locally",
				"key", key, "blockid", blockID, "err", err)

			return
		}

		writer.WriteHeader(http.StatusCreated)

		return
	}

	// Retrieve an already existing uploadable ID or compute a new one
	uploadID, err := uploadable.IDOrCompute(func() (string, error) {
		multipartCacheUploadCreateResponse, err := client.CirrusClient.MultipartCacheUploadCreate(request.Context(),
			cacheKey)
		if err != nil {
			return "", err
		}

		return multipartCacheUploadCreateResponse.UploadId, nil
	})
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "failed to create new multipart upload",
			"key", key, "blockid", blockID, "err", err)

		return
	}

	multipartCacheUploadPartResponse, err := client.CirrusClient.MultipartCacheUploadPart(request.Context(),
		&api.MultipartCacheUploadPartRequest{
			CacheKey:      cacheKey,
			UploadId:      uploadID,
			PartNumber:    uint32(partNumber),
			ContentLength: contentLength,
		},
	)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "failed to create new multipart upload part",
			"key", key, "blockid", blockID, "uploadid", uploadID, "err", err)

		return
	}

	// Proxy cache entry upload since we need an ETag
	req, err := http.NewRequestWithContext(request.Context(), http.MethodPut,
		multipartCacheUploadPartResponse.Url, request.Body)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "failed to create request to proxy "+
			"cache multipart entry upload", "key", key, "blockid", blockID, "err", err)

		return
	}

	// Content-Length is pre-signed, so we need to provide it
	req.ContentLength = int64(contentLength)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "failed to perform request to proxy "+
			"cache multipart entry upload", "key", key, "blockid", blockID, "err", err)

		return
	}

	if resp.StatusCode != http.StatusOK {
		fail(writer, request, http.StatusInternalServerError, fmt.Sprintf("failed to perform request to proxy "+
			"cache multipart entry upload, got unexpected HTTP %d", resp.StatusCode), "key", key,
			"blockid", blockID)

		return
	}

	if err := uploadable.AppendPart(partNumber, resp.Header.Get("ETag")); err != nil {
		fail(writer, request, http.StatusBadRequest, "failed to finalize part upload", "err", err)

		return
	}

	writer.WriteHeader(http.StatusCreated)
}

func (azureBlob *AzureBlob) putBlockList(writer http.ResponseWriter, request *http.Request) {
	key := request.PathValue("key")

	cacheKey := &api.CacheKey{
		TaskIdentification: client.CirrusTaskIdentification,
		CacheKey:           key,
	}

	var blockList blockList

	if err := render.DecodeXML(request.Body, &blockList); err != nil {
		fail(writer, request, http.StatusBadRequest, "failed to parse block list",
			"key", key, "err", err)

		return
	}

	if len(blockList.Committed) != 0 || len(blockList.Uncommitted) != 0 {
		fail(writer, request, http.StatusBadRequest, "only Latest blocks are supported, "+
			"got Committed/Uncommitted")

		return
	}

	uploadable, ok := azureBlob.uploadables.Load(key)
	if !ok {
		fail(writer, request, http.StatusBadRequest, "received a block list for a non-existent upload",
			"key", key)

		return
	}

	uploadID, ok := uploadable.ID()
	if !ok {
		fail(writer, request, http.StatusBadRequest, "received a block list for a non-initialized upload",
			"key", key)

		return
	}

	var localPartReaders []io.Reader
	var protoParts []*api.MultipartCacheUploadCommitRequest_Part

	for _, blockID := range blockList.Latest {
		// Decode the part number
		partNumber, err := blockIDToPartNumber(blockID)
		if err != nil {
			fail(writer, request, http.StatusBadRequest, "received a block list pointing to a non-parseable block",
				"key", key, "blockid", blockID, "err", err)

			return
		}

		part := uploadable.GetPart(partNumber)
		if part == nil {
			fail(writer, request, http.StatusBadRequest, "received a block list pointing to a non-existent block",
				"key", key, "blockid", blockID, "index", partNumber)

			return
		}

		if uploadable.Local() {
			localPartReaders = append(localPartReaders, part.File())
		} else {
			protoParts = append(protoParts, &api.MultipartCacheUploadCommitRequest_Part{
				PartNumber: partNumber,
				Etag:       part.ETag(),
			})
		}
	}

	if uploadable.Local() {
		generateCacheUploadURLResponse, err := client.CirrusClient.GenerateCacheUploadURL(request.Context(), &api.CacheKey{
			TaskIdentification: client.CirrusTaskIdentification,
			CacheKey:           key,
		})
		if err != nil {
			fail(writer, request, http.StatusInternalServerError, "failed to generate cache upload URL "+
				"for local part upload", "key", key, "uploadid", uploadID, "err", err)

			return
		}

		uploadReq, err := http.NewRequest(http.MethodPut, generateCacheUploadURLResponse.Url, io.MultiReader(localPartReaders...))
		if err != nil {
			fail(writer, request, http.StatusInternalServerError, "failed to create request to cache upload URL "+
				"for local part upload", "key", key, "uploadid", uploadID, "err", err)

			return
		}

		uploadResp, err := http.DefaultClient.Do(uploadReq)
		if err != nil {
			fail(writer, request, http.StatusInternalServerError, "failed to perform request to cache upload URL "+
				"for local part upload", "key", key, "uploadid", uploadID, "err", err)

			return
		}

		if uploadResp.StatusCode != http.StatusOK {
			fail(writer, request, http.StatusInternalServerError, "failed to create request to cache upload URL "+
				"for local part upload, got unexpected HTTP status", "key", key, "uploadid", uploadID,
				"http_status_code", uploadResp.StatusCode, "http_status", uploadResp.Status)

			return
		}

		writer.WriteHeader(http.StatusCreated)

		return
	}

	_, err := client.CirrusClient.MultipartCacheUploadCommit(request.Context(), &api.MultipartCacheUploadCommitRequest{
		CacheKey: cacheKey,
		UploadId: uploadID,
		Parts:    protoParts,
	})
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "failed to commit a multipart upload",
			"key", key, "uploadid", uploadID, "err", err)

		return
	}

	writer.WriteHeader(http.StatusCreated)
}
