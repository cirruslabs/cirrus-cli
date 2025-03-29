package http_cache

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/azureblob"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacachev2"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

const (
	activeRequestsPerLogicalCPU = 4

	CirrusHeaderCreatedBy = "Cirrus-Created-By"
)

type HTTPCache struct {
	httpClient                   *http.Client
	potentiallyCachingHTTPClient *http.Client
}

var sem = semaphore.NewWeighted(int64(runtime.NumCPU() * activeRequestsPerLogicalCPU))

func DefaultTransport() *http.Transport {
	maxConcurrentConnections := runtime.NumCPU() * activeRequestsPerLogicalCPU

	return &http.Transport{
		MaxIdleConns:        maxConcurrentConnections,
		MaxIdleConnsPerHost: maxConcurrentConnections, // default is 2 which is too small
	}
}

func Start(potentiallyCachingTransport http.RoundTripper, chachaEnabled bool, opts ...Option) string {
	httpCache := &HTTPCache{
		httpClient: &http.Client{
			Transport: DefaultTransport(),
			Timeout:   10 * time.Minute,
		},
		potentiallyCachingHTTPClient: &http.Client{
			Transport: potentiallyCachingTransport,
			Timeout:   10 * time.Minute,
		},
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
			azureblob.New(httpCache.potentiallyCachingHTTPClient, chachaEnabled))))

		// Apply options
		for _, opt := range opts {
			opt(mux)
		}

		go http.Serve(listener, mux)
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
		checkCacheExists(w, key)
	} else if r.Method == http.MethodPost {
		httpCache.uploadCacheEntry(w, r, key)
	} else if r.Method == http.MethodPut {
		httpCache.uploadCacheEntry(w, r, key)
	} else if r.Method == http.MethodDelete {
		deleteCacheEntry(w, key)
	} else {
		log.Printf("Not supported request method: %s\n", r.Method)
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
		log.Printf("%s cache info failed: %v\n", cacheKey, err)
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
		log.Printf("%s cache download failed: %v\n", cacheKey, err)

		// RPC fallback
		if status.Code(err) == codes.Unimplemented {
			log.Println("Falling back to downloading cache over RPC...")
			httpCache.downloadCacheViaRPC(w, r, cacheKey)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	} else {
		log.Printf("Redirecting cache download of %s\n", cacheKey)
		httpCache.proxyDownloadFromURLs(w, response.Urls)
	}
}

func (httpCache *HTTPCache) proxyDownloadFromURLs(w http.ResponseWriter, urls []string) {
	for _, url := range urls {
		if httpCache.proxyDownloadFromURL(w, url) {
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

func (httpCache *HTTPCache) proxyDownloadFromURL(w http.ResponseWriter, url string) bool {
	resp, err := httpCache.potentiallyCachingHTTPClient.Get(url)
	if err != nil {
		log.Printf("Proxying cache %s failed: %v\n", url, err)
		return false
	}
	defer resp.Body.Close()
	successfulStatus := 100 <= resp.StatusCode && resp.StatusCode < 300
	if !successfulStatus {
		log.Printf("Proxying cache %s failed with %d status\n", url, resp.StatusCode)
		return false
	}
	w.WriteHeader(resp.StatusCode)
	bytesRead, err := io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Proxying cache download for %s failed with %v\n", url, err)
		return false
	} else {
		log.Printf("Proxying cache %s succeded! Proxies %d bytes!\n", url, bytesRead)
	}
	return true
}

func (httpCache *HTTPCache) uploadCacheEntry(w http.ResponseWriter, r *http.Request, cacheKey string) {
	key := api.CacheKey{
		TaskIdentification: client.CirrusTaskIdentification,
		CacheKey:           cacheKey,
	}
	generateResp, err := client.CirrusClient.GenerateCacheUploadURL(context.Background(), &key)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to initialized uploading of %s cache! %s", cacheKey, err)
		log.Println(errorMsg)

		// RPC fallback
		if status.Code(err) == codes.Unimplemented {
			log.Println("Falling back to uploading cache over RPC...")
			uploadCacheEntryViaRPC(w, r, cacheKey)

			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorMsg))
		return
	}
	req, err := http.NewRequest("PUT", generateResp.Url, bufio.NewReader(r.Body))
	if err != nil {
		log.Printf("%s cache upload failed: %v\n", cacheKey, err)
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
		log.Println(errorMsg)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorMsg))
		return
	}
	if resp.StatusCode >= 400 {
		log.Printf("Failed to proxy upload of %s cache! %s", cacheKey, resp.Status)
		log.Printf("Headers for PUT request to  %s\n", generateResp.Url)
		req.Header.Write(log.Writer())
		log.Println("Failed response:")
		resp.Write(log.Writer())
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
		log.Println(errorMsg)

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(errorMsg))
		return
	}

	w.WriteHeader(http.StatusOK)
}
