package azureblob

import (
	"fmt"
	"net/http"

	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/blobstorage"
)

func (azureBlob *AzureBlob) headBlobAbstract(writer http.ResponseWriter, request *http.Request) {
	switch request.URL.Query().Get("comp") {
	default:
		azureBlob.headBlob(writer, request)
	}
}

func (azureBlob *AzureBlob) headBlob(writer http.ResponseWriter, request *http.Request) {
	key := request.PathValue("key")

	urlInfos, err := azureBlob.storage.DownloadURLs(request.Context(), key)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "failed to generate cache download URLs",
			"key", key, "err", err)

		return
	}

	if len(urlInfos) == 0 {
		fail(writer, request, http.StatusInternalServerError, fmt.Sprintf("failed to generate"+
			" cache download URLs: expected at least 1 URL, got 0"))

		return
	}

	// Retrieve cache entry information
	for i, info := range urlInfos {
		isLastIteration := i == len(urlInfos)-1

		if azureBlob.retrieveCacheEntryInfo(writer, request, key, info, isLastIteration) {
			break
		}
	}
}

func (azureBlob *AzureBlob) retrieveCacheEntryInfo(
	writer http.ResponseWriter,
	request *http.Request,
	key string,
	info *blobstorage.URLInfo,
	isLastIteration bool,
) bool {
	if info == nil {
		return false
	}

	req, err := http.NewRequestWithContext(request.Context(), http.MethodGet, info.URL, nil)
	if err != nil {
		if !isLastIteration {
			return false
		}

		fail(writer, request, http.StatusInternalServerError, "failed to create request to retrieve"+
			" cache entry information", "key", key, "err", err)

		return true
	}

	for header, value := range info.ExtraHeaders {
		req.Header.Set(header, value)
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

	writer.WriteHeader(resp.StatusCode)

	return true
}
