package azureblob

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
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

	http.Redirect(writer, request, generateCacheDownloadURLResponse.Urls[0], http.StatusTemporaryRedirect)
}
