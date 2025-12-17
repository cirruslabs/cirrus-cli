package azureblob

import (
	"fmt"
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
	urls, err := azureBlob.storageBackend.DownloadURLs(request.Context(), key)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "failed to generate cache download URLs",
			"key", key, "err", err)

		return
	}

	if len(urls) == 0 {
		fail(writer, request, http.StatusInternalServerError, fmt.Sprintf("failed to generate"+
			" cache download URLs: expected at least 1 URL, got 0"))

		return
	}

	// Retrieve cache entry information
	for i, url := range urls {
		isLastIteration := i == len(urls)-1

		if azureBlob.retrieveCacheEntryInfo(writer, request, key, url.URL, isLastIteration) {
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

	resp, err := azureBlob.httpClient.Do(req)
	if err != nil {
		if !isLastIteration {
			return false
		}

		fail(writer, request, http.StatusInternalServerError, "failed to perform request to retrieve"+
			" cache entry information", "key", key, "err", err)

		return true
	}

	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		writer.Header().Set("Content-Length", contentLength)
	}

	writer.WriteHeader(resp.StatusCode)

	return true
}
