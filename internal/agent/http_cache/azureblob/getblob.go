package azureblob

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"log/slog"

	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/azureblob/simplerange"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/azureblob/unexpectedeofreader"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/blobstorage"
	"github.com/cirruslabs/cirrus-cli/internal/agent/progressreader"
	"github.com/dustin/go-humanize"
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

	// Proxy cache entry download
	for i, info := range urlInfos {
		isLastIteration := i == len(urlInfos)-1

		if azureBlob.proxyCacheEntryDownload(writer, request, key, info, isLastIteration) {
			break
		}
	}
}

func (azureBlob *AzureBlob) proxyCacheEntryDownload(
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

		fail(writer, request, http.StatusInternalServerError, "failed to create request to proxy"+
			" cache entry download", "key", key, "err", err)

		return true
	}

	for header, value := range info.ExtraHeaders {
		req.Header.Set(header, value)
	}

	// Support HTTP range requests
	var rangeHeaderToUse string

	if rangeHeader := request.Header.Get("Range"); rangeHeader != "" {
		rangeHeaderToUse = rangeHeader
	}
	if rangeHeader := request.Header.Get("X-Ms-Range"); rangeHeader != "" {
		rangeHeaderToUse = rangeHeader
	}

	if rangeHeaderToUse != "" {
		req.Header.Set("Range", rangeHeaderToUse)
	}

	resp, err := azureBlob.httpClient.Do(req)
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

	writer.WriteHeader(resp.StatusCode)

	startProxyingAt := time.Now()
	// we usually proxy large files so let's use a larger buffer
	largeBuffer := make([]byte, PROXY_DOWNLOAD_BUFFER_SIZE)

	reader := resp.Body

	if azureBlob.withUnexpectedEOFReader {
		reader = io.NopCloser(unexpectedeofreader.New(resp.Body))
	}

	progressReader := progressreader.New(reader, PROXY_DOWNLOAD_PROGRESS_LOG_INTERVAL, func(bytes int64, duration time.Duration) {
		rate := float64(bytes) / duration.Seconds()

		log.Printf("Proxying cache entry download for %s: %d bytes read in %s (%s/s)",
			key, bytes, duration, humanize.Bytes(uint64(rate)))
	})
	bytesRead, err := io.CopyBuffer(writer, progressReader, largeBuffer)
	if err != nil {
		fail(nil, request, http.StatusInternalServerError, "failed to proxy cache entry download",
			"err", err, "duration", time.Since(startProxyingAt), "read", bytesRead, "key", key)

		// Try to recover by adjusting Range header and re-issuing the request
		if errors.Is(err, io.ErrUnexpectedEOF) {
			bytesRecovered, err := azureBlob.proxyRecover(request.Context(), rangeHeaderToUse, resp, info, bytesRead, writer)
			if err != nil {
				craftAndLogMessage(slog.LevelError, "failed to recover proxy cache entry download",
					"err", err)
			} else {
				craftAndLogMessage(slog.LevelInfo, "successfully recovered proxy cache entry download",
					"read", bytesRecovered, "key", key)
			}
		}

		return true
	}

	return true
}

func (azureBlob *AzureBlob) proxyRecover(
	ctx context.Context,
	rangeHeader string,
	upstreamResponse *http.Response,
	info *blobstorage.URLInfo,
	bytesRead int64,
	writer http.ResponseWriter,
) (int64, error) {
	var start int64
	var end *int64
	var err error

	if rangeHeader != "" {
		// Take into account the Range header specified in a downstream request
		start, end, err = simplerange.Parse(rangeHeader)
		if err != nil {
			return 0, fmt.Errorf("failed to parse Range header %q: %w", rangeHeader, err)
		}
	}

	// Retrieve an identifier from the upstream response
	// to detect possible object modification
	var ifRangeValue string

	if eTag := upstreamResponse.Header.Get("ETag"); eTag != "" {
		ifRangeValue = eTag
	} else if lastModified := upstreamResponse.Header.Get("Last-Modified"); lastModified != "" {
		ifRangeValue = lastModified
	} else {
		return 0, fmt.Errorf("no ETag or Last-Modifier header found to use for If-Range")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.URL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create an additional request: %w", err)
	}

	if end != nil {
		if start+bytesRead > *end {
			return 0, fmt.Errorf("range start + bytes read (%d) is larger than range end (%d)",
				start+bytesRead, *end)
		}

		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start+bytesRead, *end))
	} else {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", start+bytesRead))
	}

	req.Header.Set("If-Range", ifRangeValue)

	for header, value := range info.ExtraHeaders {
		req.Header.Set(header, value)
	}

	resp, err := azureBlob.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusPartialContent:
		// Proceed with proxying
	default:
		return 0, fmt.Errorf("got unexpected HTTP %d", resp.StatusCode)
	}

	return io.Copy(writer, resp.Body)
}
