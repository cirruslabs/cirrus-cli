package http_cache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/cirruslabs/cirrus-cli/internal/agent/grpcutils"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/azureblob"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacachev2"
	agentstorage "github.com/cirruslabs/cirrus-cli/internal/agent/storage"
	"github.com/cirruslabs/omni-cache/pkg/storage"
	urlproxy "github.com/cirruslabs/omni-cache/pkg/url-proxy"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	activeRequestsPerLogicalCPU = 4

	CirrusHeaderCreatedBy = "Cirrus-Created-By"
)

type HTTPCache struct {
	httpClient    *http.Client
	azureBlobOpts []azureblob.Option
	proxy         *urlproxy.Proxy
	backend       agentstorage.CacheBackend
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
	backend agentstorage.CacheBackend,
	opts ...Option,
) string {
	if backend == nil {
		panic("http_cache.Start: backend is required")
	}
	if transport == nil {
		transport = DefaultTransport()
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Minute,
	}
	proxyOptions := []urlproxy.ProxyOption{urlproxy.WithHTTPClient(httpClient)}
	if outgoingMD, ok := metadata.FromOutgoingContext(ctx); ok {
		proxyOptions = append(
			proxyOptions,
			urlproxy.WithGRPCDialOptions(
				grpc.WithUnaryInterceptor(grpcutils.UnaryMetadataInterceptor(outgoingMD)),
				grpc.WithStreamInterceptor(grpcutils.StreamMetadataInterceptor(outgoingMD)),
			),
		)
	}
	httpCache := &HTTPCache{
		httpClient: httpClient,
		proxy:      urlproxy.NewProxy(proxyOptions...),
		backend:    backend,
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
			ghacache.New(address, httpCache.backend))))

		// GitHub Actions cache API v2
		//
		// Note that we don't strip the prefix here because
		// Twirp handler inside *ghacachev2.Cache expects it.
		ghaCacheV2 := ghacachev2.New(address, httpCache.backend)
		mux.Handle(ghaCacheV2.PathPrefix(), ghaCacheV2)

		// Partial Azure Blob Service REST API implementation
		// needed for the GHA cache API v2 to function properly
		mux.Handle(azureblob.APIMountPoint+"/", sentryHandler.Handle(http.StripPrefix(azureblob.APIMountPoint,
			azureblob.New(httpCache.backend, httpCache.httpClient, httpCache.azureBlobOpts...))))

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
		httpCache.checkCacheExists(w, r, key)
	} else if r.Method == http.MethodPost {
		httpCache.uploadCacheEntry(w, r, key)
	} else if r.Method == http.MethodPut {
		httpCache.uploadCacheEntry(w, r, key)
	} else if r.Method == http.MethodDelete {
		httpCache.deleteCacheEntry(w, r, key)
	} else {
		slog.Warn("Not supported request method", "method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (httpCache *HTTPCache) checkCacheExists(w http.ResponseWriter, r *http.Request, cacheKey string) {
	info, err := httpCache.backend.CacheInfo(r.Context(), cacheKey, nil)
	if err != nil {
		slog.Error("Cache info failed", "cache_key", cacheKey, "err", err)
		w.WriteHeader(http.StatusNotFound)
	} else {
		if info.OldCreatedByTaskId > 0 {
			w.Header().Set(CirrusHeaderCreatedBy, strconv.FormatInt(info.OldCreatedByTaskId, 10))
		} else if info.CreatedByTaskId != "" {
			w.Header().Set(CirrusHeaderCreatedBy, info.CreatedByTaskId)
		}
		w.Header().Set("Content-Length", strconv.FormatInt(info.SizeInBytes, 10))
		w.WriteHeader(http.StatusOK)
	}
}

func (httpCache *HTTPCache) downloadCache(w http.ResponseWriter, r *http.Request, cacheKey string) {
	urls, err := httpCache.backend.DownloadURLs(r.Context(), cacheKey)
	if err != nil {
		slog.Error("Cache download failed", "cache_key", cacheKey, "err", err)

		w.WriteHeader(http.StatusNotFound)
	} else {
		slog.Info("Redirecting cache download", "cache_key", cacheKey)
		httpCache.proxyDownloadFromURLs(w, r, cacheKey, urls)
	}
}

func (httpCache *HTTPCache) proxyDownloadFromURLs(w http.ResponseWriter, r *http.Request, cacheKey string, urls []*storage.URLInfo) {
	for _, urlInfo := range urls {
		if httpCache.proxy.ProxyDownloadFromURL(r.Context(), w, urlInfo, cacheKey) {
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

func (httpCache *HTTPCache) uploadCacheEntry(w http.ResponseWriter, r *http.Request, cacheKey string) {
	urlInfo, err := httpCache.backend.UploadURL(r.Context(), cacheKey, nil)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to initialized uploading of %s cache! %s", cacheKey, err)
		slog.Error(errorMsg)

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorMsg))
		return
	}
	uploadResource := urlproxy.UploadResource{
		Body:          r.Body,
		ContentLength: r.ContentLength,
		ResourceName:  cacheKey,
	}
	httpCache.proxy.ProxyUploadToURL(r.Context(), w, urlInfo, uploadResource)
}

func (httpCache *HTTPCache) deleteCacheEntry(w http.ResponseWriter, r *http.Request, cacheKey string) {
	err := httpCache.backend.DeleteCache(r.Context(), cacheKey)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to delete cache entry %s: %v", cacheKey, err)
		slog.Error(errorMsg)

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorMsg))
		return
	}

	w.WriteHeader(http.StatusOK)
}
