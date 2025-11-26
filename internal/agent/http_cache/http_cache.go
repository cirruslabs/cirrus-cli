package http_cache

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/azureblob"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacachev2"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/omni-cache/pkg/storage"
	urlproxy "github.com/cirruslabs/omni-cache/pkg/url-proxy"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	activeRequestsPerLogicalCPU = 4

	CirrusHeaderCreatedBy = "Cirrus-Created-By"
)

type HTTPCache struct {
	httpClient    *http.Client
	azureBlobOpts []azureblob.Option
	proxy         *urlproxy.Proxy
}

var sem = semaphore.NewWeighted(int64(runtime.NumCPU() * activeRequestsPerLogicalCPU))

func DefaultTransport() *http.Transport {
	maxConcurrentConnections := runtime.NumCPU() * activeRequestsPerLogicalCPU

	return &http.Transport{
		MaxIdleConns:        maxConcurrentConnections,
		MaxIdleConnsPerHost: maxConcurrentConnections, // default is 2 which is too small
	}
}

func Start(
	ctx context.Context,
	transport http.RoundTripper,
	opts ...Option,
) string {
	if transport == nil {
		transport = DefaultTransport()
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Minute,
	}
	httpCache := &HTTPCache{
		httpClient: httpClient,
		proxy:      urlproxy.NewProxy(urlproxy.WithHTTPClient(httpClient)),
	}

	// Apply opts
	for _, opt := range opts {
		opt(httpCache)
	}

	mux := http.NewServeMux()

	// HTTP cache protocol
	mux.HandleFunc("/{objectname...}", httpCache.handler)

	address := "127.0.0.1:12321"
	listener, err := net.Listen("tcp", address)

	if err != nil {
		slog.Warn("Port 12321 is occupied, looking for another one", "err", err)
		listener, err = net.Listen("tcp", "127.0.0.1:0")
	}
	if err == nil {
		address = listener.Addr().String()
		slog.Info("Starting http cache server", "address", address)

		// GitHub Actions cache API
		sentryHandler := sentryhttp.New(sentryhttp.Options{})

		mux.Handle(ghacache.APIMountPoint+"/", sentryHandler.Handle(http.StripPrefix(ghacache.APIMountPoint,
			ghacache.New(address))))

		// GitHub Actions cache API v2
		//
		// Note that we don't strip the prefix here because
		// Twirp handler inside *ghacachev2.Cache expects it.
		ghaCacheV2 := ghacachev2.New(address)
		mux.Handle(ghaCacheV2.PathPrefix(), ghaCacheV2)

		// Partial Azure Blob Service REST API implementation
		// needed for the GHA cache API v2 to function properly
		mux.Handle(azureblob.APIMountPoint+"/", sentryHandler.Handle(http.StripPrefix(azureblob.APIMountPoint,
			azureblob.New(httpCache.httpClient, httpCache.azureBlobOpts...))))

		httpServer := &http.Server{
			// Use agent's context as a base for the HTTP cache handlers
			BaseContext: func(_ net.Listener) context.Context {
				return ctx
			},
			Handler: mux,
		}

		go httpServer.Serve(listener)
	} else {
		slog.Error("Failed to start http cache server", "address", address, "err", err)
	}
	return address
}

func (httpCache *HTTPCache) handler(w http.ResponseWriter, r *http.Request) {
	// Limit request concurrency
	if err := sem.Acquire(r.Context(), 1); err != nil {
		slog.Warn("Failed to acquire the semaphore", "err", err)
		if errors.Is(err, context.Canceled) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			w.WriteHeader(http.StatusRequestTimeout)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		sem.Release(1)
	}()

	key := r.URL.Path
	if key[0] == '/' {
		key = key[1:]
	}
	if len(key) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if r.Method == http.MethodGet {
		httpCache.downloadCache(w, r, key)
	} else if r.Method == http.MethodHead {
		checkCacheExists(w, key)
	} else if r.Method == http.MethodPost {
		httpCache.uploadCacheEntry(w, r, key)
	} else if r.Method == http.MethodPut {
		httpCache.uploadCacheEntry(w, r, key)
	} else if r.Method == http.MethodDelete {
		deleteCacheEntry(w, key)
	} else {
		slog.Warn("Not supported request method", "method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func checkCacheExists(w http.ResponseWriter, cacheKey string) {
	cacheInfoRequest := api.CacheInfoRequest{
		TaskIdentification: client.CirrusTaskIdentification,
		CacheKey:           cacheKey,
	}
	response, err := client.CirrusClient.CacheInfo(context.Background(), &cacheInfoRequest)
	if err != nil {
		slog.Error("Cache info failed", "cache_key", cacheKey, "err", err)
		w.WriteHeader(http.StatusNotFound)
	} else {
		if response.Info.OldCreatedByTaskId > 0 {
			w.Header().Set(CirrusHeaderCreatedBy, strconv.FormatInt(response.Info.OldCreatedByTaskId, 10))
		} else if response.Info.CreatedByTaskId != "" {
			w.Header().Set(CirrusHeaderCreatedBy, response.Info.CreatedByTaskId)
		}
		w.Header().Set("Content-Length", strconv.FormatInt(response.Info.SizeInBytes, 10))
		w.WriteHeader(http.StatusOK)
	}
}

func (httpCache *HTTPCache) downloadCache(w http.ResponseWriter, r *http.Request, cacheKey string) {
	key := api.CacheKey{
		TaskIdentification: client.CirrusTaskIdentification,
		CacheKey:           cacheKey,
	}
	response, err := client.CirrusClient.GenerateCacheDownloadURLs(context.Background(), &key)
	if err != nil {
		slog.Error("Cache download failed", "cache_key", cacheKey, "err", err)

		// RPC fallback
		if status.Code(err) == codes.Unimplemented {
			slog.Info("Falling back to downloading cache over RPC...")
			httpCache.downloadCacheViaRPC(w, r, cacheKey)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	} else {
		slog.Info("Redirecting cache download", "cache_key", cacheKey)
		httpCache.proxyDownloadFromURLs(w, r, cacheKey, response.Urls)
	}
}

func (httpCache *HTTPCache) proxyDownloadFromURLs(w http.ResponseWriter, r *http.Request, cacheKey string, urls []string) {
	for _, url := range urls {
		urlInfo := storage.URLInfo{URL: url}
		if httpCache.proxy.ProxyDownloadFromURL(r.Context(), w, &urlInfo, cacheKey) {
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

func (httpCache *HTTPCache) uploadCacheEntry(w http.ResponseWriter, r *http.Request, cacheKey string) {
	key := api.CacheKey{
		TaskIdentification: client.CirrusTaskIdentification,
		CacheKey:           cacheKey,
	}
	generateResp, err := client.CirrusClient.GenerateCacheUploadURL(context.Background(), &key)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to initialized uploading of %s cache! %s", cacheKey, err)
		slog.Error(errorMsg)

		// RPC fallback
		if status.Code(err) == codes.Unimplemented {
			slog.Info("Falling back to uploading cache over RPC...")
			uploadCacheEntryViaRPC(w, r, cacheKey)

			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorMsg))
		return
	}
	req, err := http.NewRequest("PUT", generateResp.Url, bufio.NewReader(r.Body))
	if err != nil {
		slog.Error("Cache upload failed", "cache_key", cacheKey, "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = r.ContentLength
	for k, v := range generateResp.GetExtraHeaders() {
		req.Header.Set(k, v)
	}
	resp, err := httpCache.httpClient.Do(req)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to proxy upload of %s cache! %s", cacheKey, err)
		slog.Error(errorMsg)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorMsg))
		return
	}
	if resp.StatusCode >= 400 {
		slog.Error("Failed to proxy upload of cache", "cache_key", cacheKey, "status", resp.Status)

		var headersBuilder strings.Builder
		req.Header.Write(&headersBuilder)
		slog.Error("Headers for PUT request", "url", generateResp.Url, "headers", headersBuilder.String())

		var responseBuilder strings.Builder
		resp.Write(&responseBuilder)
		slog.Error("Failed response", "response", responseBuilder.String())
	}
	w.WriteHeader(resp.StatusCode)
}

func deleteCacheEntry(w http.ResponseWriter, cacheKey string) {
	request := api.DeleteCacheRequest{
		TaskIdentification: client.CirrusTaskIdentification,
		CacheKey:           cacheKey,
	}

	_, err := client.CirrusClient.DeleteCache(context.Background(), &request)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to delete cache entry %s: %v", cacheKey, err)
		slog.Error(errorMsg)

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorMsg))
		return
	}

	w.WriteHeader(http.StatusOK)
}
