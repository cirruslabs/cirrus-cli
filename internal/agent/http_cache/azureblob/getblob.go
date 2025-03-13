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
		fail(writer, request, http.StatusInternalServerError, "failed to generate cache download URL",
			"key", key, "err", err)

		return
	}

	if len(generateCacheDownloadURLResponse.Urls) != 1 {
		fail(writer, request, http.StatusInternalServerError, fmt.Sprintf("failed to generate"+
			" cache download URL: expected 1 URL, got %d", len(generateCacheDownloadURLResponse.Urls)))

		return
	}

	// Proxy cache entry download
	req, err := http.NewRequestWithContext(request.Context(), http.MethodGet,
		generateCacheDownloadURLResponse.Urls[0], nil)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "failed to create request to proxy"+
			" cache entry download", "key", key, "err", err)

		return
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
		fail(writer, request, http.StatusInternalServerError, "failed to perform request to proxy"+
			" cache entry download", "key", key, "err", err)

		return
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// proceed with proxying
	case http.StatusNotFound:
		writer.WriteHeader(http.StatusNotFound)

		return
	default:
		fail(writer, request, http.StatusInternalServerError, fmt.Sprintf("failed to perform request to proxy"+
			" cache entry download, got unexpected HTTP %d", resp.StatusCode), "key", key)

		return
	}

	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		writer.Header().Set("Content-Length", contentLength)
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
		proxyingDuration := time.Since(startProxyingAt)
		fail(writer, request, http.StatusInternalServerError, "failed to proxy cache entry download",
			"err", err, "duration", proxyingDuration, "read", bytesRead, "key", key)
		return
	}
}
