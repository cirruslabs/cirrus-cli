package azureblob

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"net/http"
)

func (azureBlob *AzureBlob) headBlobAbstract(writer http.ResponseWriter, request *http.Request) {
	switch request.URL.Query().Get("comp") {
	default:
		azureBlob.headBlob(writer, request)
	}
}

func (azureBlob *AzureBlob) headBlob(writer http.ResponseWriter, request *http.Request) {
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

	// Retrieve cache entry information
	req, err := http.NewRequestWithContext(request.Context(), http.MethodGet,
		generateCacheDownloadURLResponse.Urls[0], nil)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "failed to create request to retrieve"+
			" cache entry information", "key", key, "err", err)

		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "failed to perform request to retrieve"+
			" cache entry information", "key", key, "err", err)

		return
	}

	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		writer.Header().Set("Content-Length", contentLength)
	}

	writer.WriteHeader(resp.StatusCode)
}
