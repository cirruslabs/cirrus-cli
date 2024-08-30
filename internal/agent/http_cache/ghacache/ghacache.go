package ghacache

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache/httprange"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache/uploadable"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/go-chi/render"
	"github.com/puzpuzpuz/xsync/v3"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	APIMountPoint = "/_apis/artifactcache"

	// JavaScript's Number is limited to 2^53-1[1]
	//
	// [1]: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Number/MAX_SAFE_INTEGER
	jsNumberMaxSafeInteger = 9007199254740991
)

type GHACache struct {
	cacheHost   string
	mux         *http.ServeMux
	uploadables *xsync.MapOf[int64, *uploadable.Uploadable]
}

func New(cacheHost string) *GHACache {
	cache := &GHACache{
		cacheHost:   cacheHost,
		mux:         http.NewServeMux(),
		uploadables: xsync.NewMapOf[int64, *uploadable.Uploadable](),
	}

	cache.mux.HandleFunc("GET /cache", cache.get)
	cache.mux.HandleFunc("POST /caches", cache.reserveUploadable)
	cache.mux.HandleFunc("PATCH /caches/{id}", cache.updateUploadable)
	cache.mux.HandleFunc("POST /caches/{id}", cache.commitUploadable)

	return cache
}

func (cache *GHACache) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	cache.mux.ServeHTTP(writer, request)
}

func (cache *GHACache) get(writer http.ResponseWriter, request *http.Request) {
	keys := strings.Split(request.URL.Query().Get("keys"), ",")
	version := request.URL.Query().Get("version")

	for _, key := range keys {
		httpCacheURL := cache.httpCacheURL(key, version)

		resp, err := http.Head(httpCacheURL)
		if err != nil {
			fail(writer, request, http.StatusInternalServerError, "GHA cache failed to "+
				"retrieve %q: %v", httpCacheURL, err)

			return
		}

		if resp.StatusCode == http.StatusOK {
			jsonResp := struct {
				Key string `json:"cacheKey"`
				URL string `json:"archiveLocation"`
			}{
				Key: key,
				URL: httpCacheURL,
			}

			render.JSON(writer, request, &jsonResp)

			return
		}
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (cache *GHACache) reserveUploadable(writer http.ResponseWriter, request *http.Request) {
	var jsonReq struct {
		Key     string `json:"key"`
		Version string `json:"version"`
	}

	if err := render.DecodeJSON(request.Body, &jsonReq); err != nil {
		fail(writer, request, http.StatusBadRequest, "GHA cache failed to read/decode the "+
			"JSON passed to the reserve uploadable endpoint: %v", err)

		return
	}

	jsonResp := struct {
		CacheID int64 `json:"cacheId"`
	}{
		CacheID: rand.Int63n(jsNumberMaxSafeInteger),
	}

	grpcResp, err := client.CirrusClient.MultipartCacheUploadCreate(request.Context(), &api.CacheKey{
		TaskIdentification: client.CirrusTaskIdentification,
		CacheKey:           httpCacheKey(jsonReq.Key, jsonReq.Version),
	})
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed to create "+
			"multipart upload for key %q and version %q: %v", jsonReq.Key, jsonReq.Version, err)

		return
	}

	cache.uploadables.Store(jsonResp.CacheID, uploadable.New(jsonReq.Key, jsonReq.Version, grpcResp.UploadId))

	render.JSON(writer, request, &jsonResp)
}

func (cache *GHACache) updateUploadable(writer http.ResponseWriter, request *http.Request) {
	id, ok := getID(request)
	if !ok {
		fail(writer, request, http.StatusBadRequest, "GHA cache failed to get/decode the "+
			"ID passed to the update uploadable endpoint")

		return
	}

	uploadable, ok := cache.uploadables.Load(id)
	if !ok {
		fail(writer, request, http.StatusNotFound, "GHA cache failed to find an uploadable "+
			"with ID %d", id)

		return
	}

	httpRanges, err := httprange.ParseRange(request.Header.Get("Content-Range"), math.MaxInt64)
	if err != nil {
		fail(writer, request, http.StatusBadRequest, "GHA cache failed to parse Content-Range header %q: %v",
			request.Header.Get("Content-Range"), err)

		return
	}

	if len(httpRanges) != 1 {
		fail(writer, request, http.StatusBadRequest, "GHA cache expected exactly one Content-Range value, got %d",
			len(httpRanges))

		return
	}

	partNumber, err := uploadable.RangeToPart.Tell(request.Context(), httpRanges[0].Start, httpRanges[0].Length)
	if err != nil {
		fail(writer, request, http.StatusBadRequest, "GHA cache failed to tell the part number for "+
			"Content-Range header %q: %v", request.Header.Get("Content-Range"), err)

		return
	}

	response, err := client.CirrusClient.MultipartCacheUploadPart(request.Context(), &api.MultipartCacheUploadPartRequest{
		CacheKey: &api.CacheKey{
			TaskIdentification: client.CirrusTaskIdentification,
			CacheKey:           httpCacheKey(uploadable.Key(), uploadable.Version()),
		},
		UploadId:      uploadable.UploadID(),
		PartNumber:    uint32(partNumber),
		ContentLength: uint64(httpRanges[0].Length),
	})
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed create pre-signed "+
			"upload part URL for key %q, version %q and part %d: %v", uploadable.Key(),
			uploadable.Version(), partNumber, err)

		return
	}

	uploadPartRequest, err := http.NewRequest(http.MethodPut, response.Url, request.Body)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed to create upload part "+
			"request for key %q, version %q and part %d: %v", uploadable.Key(), uploadable.Version(), partNumber, err)

		return
	}

	// Golang's HTTP client won't send the Content-Length
	// because it doesn't know the request.Body's size,
	// so we do it for it, otherwise we'll get HTTP 403
	// from S3 since non-existent "Content-Length" header
	// is not equal to "Content-Length: X" that was pre-signed.
	uploadPartRequest.ContentLength = httpRanges[0].Length

	// Add headers mandated by the pre-signed request
	for key, value := range response.ExtraHeaders {
		uploadPartRequest.Header.Set(key, value)
	}

	uploadPartResponse, err := http.DefaultClient.Do(uploadPartRequest)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed to upload part "+
			"for key %q, version %q and part %d: %v", uploadable.Key(), uploadable.Version(), partNumber, err)

		return
	}

	if uploadPartResponse.StatusCode != http.StatusOK {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed to upload part "+
			"for key %q, version %q and part %d: got HTTP %d", uploadable.Key(), uploadable.Version(), partNumber,
			uploadPartResponse.StatusCode)

		return
	}

	err = uploadable.AppendPart(uint32(partNumber), uploadPartResponse.Header.Get("ETag"), httpRanges[0].Length)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed to append part "+
			"for key %q, version %q and part %d: %v", uploadable.Key(), uploadable.Version(), partNumber, err)

		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (cache *GHACache) commitUploadable(writer http.ResponseWriter, request *http.Request) {
	id, ok := getID(request)
	if !ok {
		fail(writer, request, http.StatusBadRequest, "GHA cache failed to get/decode the "+
			"ID passed to the commit uploadable endpoint")

		return
	}

	uploadable, ok := cache.uploadables.Load(id)
	if !ok {
		fail(writer, request, http.StatusNotFound, "GHA cache failed to find an uploadable "+
			"with ID %d", id)

		return
	}
	defer cache.uploadables.Delete(id)

	var jsonReq struct {
		Size int64 `json:"size"`
	}

	if err := render.DecodeJSON(request.Body, &jsonReq); err != nil {
		fail(writer, request, http.StatusBadRequest, "GHA cache failed to read/decode "+
			"the JSON passed to the commit uploadable endpoint: %v", err)

		return
	}

	parts, partsSize, err := uploadable.Finalize()
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed to "+
			"finalize uploadable %d: %v", id, err)

		return
	}

	if jsonReq.Size != partsSize {
		fail(writer, request, http.StatusBadRequest, "GHA cache detected a cache entry "+
			"size mismatch for uploadable %d: expected %d bytes, got %d bytes",
			id, partsSize, jsonReq.Size)

		return
	}

	_, err = client.CirrusClient.MultipartCacheUploadCommit(request.Context(), &api.MultipartCacheUploadCommitRequest{
		CacheKey: &api.CacheKey{
			TaskIdentification: client.CirrusTaskIdentification,
			CacheKey:           httpCacheKey(uploadable.Key(), uploadable.Version()),
		},
		UploadId: uploadable.UploadID(),
		Parts:    parts,
	})
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed to commit multipart upload "+
			"for key %q, version %q and uploadable %q: %v", uploadable.Key(), uploadable.Version(),
			uploadable.UploadID(), err)

		return
	}

	writer.WriteHeader(http.StatusCreated)
}

func httpCacheKey(key string, version string) string {
	return fmt.Sprintf("%s-%s", url.PathEscape(key), url.PathEscape(version))
}

func (cache *GHACache) httpCacheURL(key string, version string) string {
	return fmt.Sprintf("http://%s/%s", cache.cacheHost, httpCacheKey(key, version))
}

func getID(request *http.Request) (int64, bool) {
	idRaw := request.PathValue("id")

	id, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil {
		return 0, false
	}

	return id, true
}

func fail(writer http.ResponseWriter, request *http.Request, status int, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)

	log.Println(message)

	writer.WriteHeader(status)
	jsonResp := struct {
		Message string `json:"message"`
	}{
		Message: message,
	}
	render.JSON(writer, request, &jsonResp)
}
