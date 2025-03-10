package azureblob

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
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

	resp, err := http.DefaultClient.Do(req)
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

	startProxyingAt := time.Now()
	progressLoggedAt := startProxyingAt
	// we usually proxy large files so let's use a larger buffer
	largeBuffer := make([]byte, PROXY_DOWNLOAD_BUFFER_SIZE)
	bytesRead := int64(0)
	for {
		n, proxyErr := resp.Body.Read(largeBuffer)
		if n > 0 {
			if _, writeErr := writer.Write(largeBuffer[:n]); writeErr != nil {
				proxyErr = writeErr
				break
			}
			bytesRead += int64(n)
		}
		if proxyErr == io.EOF {
			proxyErr = nil
			break
		}
		if proxyErr != nil {
			proxyingDuration := time.Since(startProxyingAt)
			fail(writer, request, http.StatusInternalServerError, "failed to proxy cache entry download",
				"err", proxyErr, "duration", proxyingDuration, "read", bytesRead, "key", key)
			break
		}
		if time.Since(progressLoggedAt) > PROXY_DOWNLOAD_PROGRESS_LOG_INTERVAL {
			currentSpeed := float64(bytesRead) / 1024 / 1024 / time.Since(startProxyingAt).Seconds()
			log.Printf("Proxying cache entry download for %s: %d bytes read in %s (avg speed %.2f Mb/s)\n", key, bytesRead, time.Since(startProxyingAt), currentSpeed)
			progressLoggedAt = time.Now()
		}
	}
}
