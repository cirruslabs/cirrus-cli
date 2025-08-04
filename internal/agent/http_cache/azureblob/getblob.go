package azureblob

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/internal/agent/progressreader"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/dustin/go-humanize"
	"io"
	"log"
	"net/http"
	"time"
)

const PROXY_DOWNLOAD_BUFFER_SIZE = 1024 * 1024
const PROXY_DOWNLOAD_PROGRESS_LOG_INTERVAL = 60 * time.Second

func (azureBlob *AzureBlob) getBlobAbstract(writer http.ResponseWriter, request *http.Request) {
	switch request.URL.Query().Get("comp") {
	default:
		azureBlob.getBlob(writer, request)
	}
}

func (azureBlob *AzureBlob) getBlob(writer http.ResponseWriter, request *http.Request) {
	key := request.PathValue("key")

	// Generate cache entry download URL
	generateCacheDownloadURLResponse, err := client.CirrusClient.GenerateCacheDownloadURLs(
		request.Context(),
		&api.CacheKey{
			TaskIdentification: client.CirrusTaskIdentification,
			CacheKey:           key,
		},
	)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "failed to generate cache download URLs",
			"key", key, "err", err)

		return
	}

	if len(generateCacheDownloadURLResponse.Urls) == 0 {
		fail(writer, request, http.StatusInternalServerError, fmt.Sprintf("failed to generate"+
			" cache download URLs: expected at least 1 URL, got 0"))

		return
	}

	// Proxy cache entry download
	for i, url := range generateCacheDownloadURLResponse.Urls {
		isLastIteration := i == len(generateCacheDownloadURLResponse.Urls)-1

		if azureBlob.proxyCacheEntryDownload(writer, request, key, url, isLastIteration) {
			break
		}
	}
}

func (azureBlob *AzureBlob) proxyCacheEntryDownload(
	writer http.ResponseWriter,
	request *http.Request,
	key string,
	url string,
	isLastIteration bool,
) bool {
	req, err := http.NewRequestWithContext(request.Context(), http.MethodGet, url, nil)
	if err != nil {
		if !isLastIteration {
			return false
		}

		fail(writer, request, http.StatusInternalServerError, "failed to create request to proxy"+
			" cache entry download", "key", key, "err", err)

		return true
	}

	// Support HTTP range requests
	if rangeHeader := request.Header.Get("Range"); rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}
	if rangeHeader := request.Header.Get("X-Ms-Range"); rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}

	resp, err := azureBlob.potentiallyCachingHTTPClient.Do(req)
	if err != nil {
		if !isLastIteration {
			return false
		}

		fail(writer, request, http.StatusInternalServerError, "failed to perform request to proxy"+
			" cache entry download", "key", key, "err", err)

		return true
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusPartialContent:
		// Proceed with proxying
	case http.StatusNotFound:
		if !isLastIteration {
			return false
		}

		writer.WriteHeader(http.StatusNotFound)

		return true
	default:
		if !isLastIteration {
			return false
		}

		fail(writer, request, http.StatusInternalServerError, fmt.Sprintf("failed to perform request to proxy"+
			" cache entry download, got unexpected HTTP %d", resp.StatusCode), "key", key)

		return true
	}

	// Chacha doesn't support Range requests yet, and not disclosing Content-Length
	// to the Azure Blob Client makes it not use the Range requests
	if !azureBlob.chachaEnabled {
		if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
			writer.Header().Set("Content-Length", contentLength)
		}
	}

	writer.WriteHeader(resp.StatusCode)

	startProxyingAt := time.Now()
	// we usually proxy large files so let's use a larger buffer
	largeBuffer := make([]byte, PROXY_DOWNLOAD_BUFFER_SIZE)
	progressReader := progressreader.New(resp.Body, PROXY_DOWNLOAD_PROGRESS_LOG_INTERVAL, func(bytes int64, duration time.Duration) {
		rate := float64(bytes) / duration.Seconds()

		log.Printf("Proxying cache entry download for %s: %d bytes read in %s (%s/s)",
			key, bytes, duration, humanize.Bytes(uint64(rate)))
	})
	bytesRead, err := io.CopyBuffer(writer, progressReader, largeBuffer)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "failed to proxy cache entry download",
			"err", err, "duration", time.Since(startProxyingAt), "read", bytesRead, "key", key)

		return true
	}

	return true
}
