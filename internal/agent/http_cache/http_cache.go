package http_cache

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/azureblob"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/blobstorage"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacachev2"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"golang.org/x/sync/semaphore"
)

const (
	activeRequestsPerLogicalCPU = 4

	CirrusHeaderCreatedBy = "Cirrus-Created-By"
)

type HTTPCache struct {
	httpClient    *http.Client
	blobStorage   blobstorage.BlobStorageBacked
	azureBlobOpts []azureblob.Option
}

var sem = semaphore.NewWeighted(int64(runtime.NumCPU() * activeRequestsPerLogicalCPU))

func DefaultTransport() *http.Transport {
	maxConcurrentConnections := runtime.NumCPU() * activeRequestsPerLogicalCPU

	return &http.Transport{
		MaxIdleConns:        maxConcurrentConnections,
		MaxIdleConnsPerHost: maxConcurrentConnections, // default is 2 which is too small
	}
}

func Start(ctx context.Context, storage blobstorage.BlobStorageBacked, opts ...Option) string {
	if storage == nil {
		log.Panic("blobStorage must not be nil when starting HTTP cache")
	}

	httpCache := &HTTPCache{
		httpClient: &http.Client{
			Transport: DefaultTransport(),
			Timeout:   10 * time.Minute,
		},
		blobStorage: storage,
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
		log.Printf("Port 12321 is occupied: %s. Looking for another one...\n", err)
		listener, err = net.Listen("tcp", "127.0.0.1:0")
	}
	if err == nil {
		address = listener.Addr().String()
		log.Printf("Starting http cache server %s\n", address)

		// GitHub Actions cache API
		sentryHandler := sentryhttp.New(sentryhttp.Options{})

		mux.Handle(ghacache.APIMountPoint+"/", sentryHandler.Handle(http.StripPrefix(ghacache.APIMountPoint,
			ghacache.New(address, storage))))

		// GitHub Actions cache API v2
		//
		// Note that we don't strip the prefix here because
		// Twirp handler inside *ghacachev2.Cache expects it.
		ghaCacheV2 := ghacachev2.New(address, storage)
		mux.Handle(ghaCacheV2.PathPrefix(), ghaCacheV2)

		// Partial Azure Blob Service REST API implementation
		// needed for the GHA cache API v2 to function properly
		mux.Handle(azureblob.APIMountPoint+"/", sentryHandler.Handle(http.StripPrefix(azureblob.APIMountPoint,
			azureblob.New(httpCache.httpClient, storage, httpCache.azureBlobOpts...))))

		httpServer := &http.Server{
			Handler: mux,
		}

		go httpServer.Serve(listener)
	} else {
		log.Printf("Failed to start http cache server %s: %s\n", address, err)
	}
	return address
}

func (httpCache *HTTPCache) handler(w http.ResponseWriter, r *http.Request) {
	// Limit request concurrency
	if err := sem.Acquire(r.Context(), 1); err != nil {
		log.Printf("Failed to acquite the semaphore: %s\n", err)
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
		httpCache.checkCacheExists(w, key)
	} else if r.Method == http.MethodPost {
		httpCache.uploadCacheEntry(w, r, key)
	} else if r.Method == http.MethodPut {
		httpCache.uploadCacheEntry(w, r, key)
	} else if r.Method == http.MethodDelete {
		httpCache.deleteCacheEntry(w, key)
	} else {
		log.Printf("Not supported request method: %s\n", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (httpCache *HTTPCache) checkCacheExists(w http.ResponseWriter, cacheKey string) {
	info, err := httpCache.blobStorage.Info(context.Background(), cacheKey, nil)
	if err != nil {
		log.Printf("%s cache info failed: %v\n", cacheKey, err)
		w.WriteHeader(http.StatusNotFound)
	} else {
		if info == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if info.OldCreatedByTaskID > 0 {
			w.Header().Set(CirrusHeaderCreatedBy, strconv.FormatInt(info.OldCreatedByTaskID, 10))
		} else if info.CreatedByTaskID != "" {
			w.Header().Set(CirrusHeaderCreatedBy, info.CreatedByTaskID)
		}

		w.Header().Set("Content-Length", strconv.FormatInt(info.SizeInBytes, 10))
		w.WriteHeader(http.StatusOK)
	}
}

func (httpCache *HTTPCache) downloadCache(w http.ResponseWriter, r *http.Request, cacheKey string) {
	urlInfos, err := httpCache.blobStorage.DownloadURLs(context.Background(), cacheKey)
	if err != nil {
		log.Printf("%s cache download failed: %v\n", cacheKey, err)

		w.WriteHeader(http.StatusNotFound)
	} else {
		log.Printf("Redirecting cache download of %s\n", cacheKey)
		httpCache.proxyDownloadFromURLs(w, r, urlInfos)
	}
}

func (httpCache *HTTPCache) proxyDownloadFromURLs(w http.ResponseWriter, r *http.Request, infos []*blobstorage.URLInfo) {
	for _, info := range infos {
		if httpCache.proxyDownloadFromURL(w, r, info) {
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

func (httpCache *HTTPCache) proxyDownloadFromURL(w http.ResponseWriter, r *http.Request, info *blobstorage.URLInfo) bool {
	if info == nil {
		return false
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, info.URL, nil)
	if err != nil {
		log.Printf("Failed to create a new GET HTTP request to URL %s: %v", info.URL, err)
		return false
	}

	for key, value := range info.ExtraHeaders {
		req.Header.Set(key, value)
	}

	resp, err := httpCache.httpClient.Do(req)
	if err != nil {
		log.Printf("Proxying cache %s failed: %v\n", info.URL, err)
		return false
	}
	defer resp.Body.Close()
	successfulStatus := 100 <= resp.StatusCode && resp.StatusCode < 300
	if !successfulStatus {
		log.Printf("Proxying cache %s failed with %d status\n", info.URL, resp.StatusCode)
		return false
	}
	w.WriteHeader(resp.StatusCode)
	bytesRead, err := io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Proxying cache download for %s failed with %v\n", info.URL, err)
		return false
	} else {
		log.Printf("Proxying cache %s succeded! Proxies %d bytes!\n", info.URL, bytesRead)
	}
	return true
}

func (httpCache *HTTPCache) uploadCacheEntry(w http.ResponseWriter, r *http.Request, cacheKey string) {
	urlInfo, err := httpCache.blobStorage.UploadURL(context.Background(), cacheKey, nil)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to initialized uploading of %s cache! %s", cacheKey, err)
		log.Println(errorMsg)

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorMsg))
		return
	}
	req, err := http.NewRequest("PUT", urlInfo.URL, bufio.NewReader(r.Body))
	if err != nil {
		log.Printf("%s cache upload failed: %v\n", cacheKey, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = r.ContentLength
	for k, v := range urlInfo.ExtraHeaders {
		req.Header.Set(k, v)
	}
	resp, err := httpCache.httpClient.Do(req)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to proxy upload of %s cache! %s", cacheKey, err)
		log.Println(errorMsg)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorMsg))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("Failed to proxy upload of %s cache! %s", cacheKey, resp.Status)
		log.Printf("Headers for PUT request to  %s\n", urlInfo.URL)
		req.Header.Write(log.Writer())
		log.Println("Failed response:")
		resp.Write(log.Writer())

		w.WriteHeader(resp.StatusCode)

		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (httpCache *HTTPCache) deleteCacheEntry(w http.ResponseWriter, cacheKey string) {
	if err := httpCache.blobStorage.Delete(context.Background(), cacheKey); err != nil {
		errorMsg := fmt.Sprintf("Failed to delete cache entry %s: %v", cacheKey, err)
		log.Println(errorMsg)

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorMsg))
		return
	}

	w.WriteHeader(http.StatusOK)
}
