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
		fail(writer, request, http.StatusInternalServerError, "failed to generate cache download URLs",
			"key", key, "err", err)

		return
	}

	if len(generateCacheDownloadURLResponse.Urls) == 0 {
		fail(writer, request, http.StatusInternalServerError, fmt.Sprintf("failed to generate"+
			" cache download URLs: expected at least 1 URL, got %d", len(generateCacheDownloadURLResponse.Urls)))

		return
	}

	// Retrieve cache entry information
	for i, url := range generateCacheDownloadURLResponse.Urls {
		isLastIteration := i == len(generateCacheDownloadURLResponse.Urls)-1

		if azureBlob.retrieveCacheEntryInfo(writer, request, key, url, isLastIteration) {
			break
		}
	}
}

func (azureBlob *AzureBlob) retrieveCacheEntryInfo(
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

		fail(writer, request, http.StatusInternalServerError, "failed to create request to retrieve"+
			" cache entry information", "key", key, "err", err)

		return true
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if !isLastIteration {
			return false
		}

		fail(writer, request, http.StatusInternalServerError, "failed to perform request to retrieve"+
			" cache entry information", "key", key, "err", err)

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

	return true
}
