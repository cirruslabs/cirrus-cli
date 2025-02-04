package azureblob

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"io"
	"net/http"
)

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

	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "failed to proxy cache entry download",
			"key", key, "err", err)

		return
	}
}
