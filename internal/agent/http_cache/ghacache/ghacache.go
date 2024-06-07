package ghacache

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache/uploadable"
	"github.com/go-chi/render"
	"github.com/puzpuzpuz/xsync/v3"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	APIMountPoint = "/_apis/artifactcache"

	// JavaScript's Number is limited to 2^53-1[1]
	//
	// [1]: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Number/MAX_SAFE_INTEGER
	jsNumberMaxSafeInteger = 9007199254740991
)

type GHACache struct {
	cacheHost   string
	mux         *http.ServeMux
	uploadables *xsync.MapOf[int64, *uploadable.Uploadable]
}

func New(cacheHost string) *GHACache {
	cache := &GHACache{
		cacheHost:   cacheHost,
		mux:         http.NewServeMux(),
		uploadables: xsync.NewMapOf[int64, *uploadable.Uploadable](),
	}

	cache.mux.HandleFunc("GET /cache", cache.get)
	cache.mux.HandleFunc("POST /caches", cache.reserveUploadable)
	cache.mux.HandleFunc("PATCH /caches/{id}", cache.updateUploadable)
	cache.mux.HandleFunc("POST /caches/{id}", cache.commitUploadable)

	return cache
}

func (cache *GHACache) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	cache.mux.ServeHTTP(writer, request)
}

func (cache *GHACache) get(writer http.ResponseWriter, request *http.Request) {
	keys := strings.Split(request.URL.Query().Get("keys"), ",")
	version := request.URL.Query().Get("version")

	for _, key := range keys {
		httpCacheURL := cache.httpCacheURL(key, version)

		resp, err := http.Head(httpCacheURL)
		if err != nil {
			fail(writer, request, http.StatusInternalServerError, "GHA cache failed to "+
				"retrieve %q: %v", httpCacheURL, err)

			return
		}

		if resp.StatusCode == http.StatusOK {
			jsonResp := struct {
				Key string `json:"cacheKey"`
				URL string `json:"archiveLocation"`
			}{
				Key: key,
				URL: httpCacheURL,
			}

			render.JSON(writer, request, &jsonResp)

			return
		}
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (cache *GHACache) reserveUploadable(writer http.ResponseWriter, request *http.Request) {
	var jsonReq struct {
		Key     string `json:"key"`
		Version string `json:"version"`
	}

	if err := render.DecodeJSON(request.Body, &jsonReq); err != nil {
		fail(writer, request, http.StatusBadRequest, "GHA cache failed to read/decode the "+
			"JSON passed to the reserve uploadable endpoint: %v", err)

		return
	}

	jsonResp := struct {
		CacheID int64 `json:"cacheId"`
	}{
		CacheID: rand.Int63n(jsNumberMaxSafeInteger),
	}

	uploadable, err := uploadable.New(jsonReq.Key, jsonReq.Version)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed instantiate "+
			"an uploadable: %v", err)

		return
	}

	cache.uploadables.Store(jsonResp.CacheID, uploadable)

	render.JSON(writer, request, &jsonResp)
}

func (cache *GHACache) updateUploadable(writer http.ResponseWriter, request *http.Request) {
	id, ok := getID(request)
	if !ok {
		fail(writer, request, http.StatusBadRequest, "GHA cache failed to get/decode the "+
			"ID passed to the update uploadable endpoint")

		return
	}

	uploadable, ok := cache.uploadables.Load(id)
	if !ok {
		fail(writer, request, http.StatusNotFound, "GHA cache failed to find an uploadable "+
			"with ID %d", id)

		return
	}

	bodyBytes, err := io.ReadAll(request.Body)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed to read a "+
			"chunk from the user for the uploadable %d: %v", id, err)

		return
	}

	if err := uploadable.WriteChunk(request.Header.Get("Content-Range"), bodyBytes); err != nil {
		fail(writer, request, http.StatusBadRequest, "GHA cache failed to write a chunk to "+
			"the uploadable %d: %v", id, err)

		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (cache *GHACache) commitUploadable(writer http.ResponseWriter, request *http.Request) {
	id, ok := getID(request)
	if !ok {
		fail(writer, request, http.StatusBadRequest, "GHA cache failed to get/decode the "+
			"ID passed to the commit uploadable endpoint")

		return
	}

	uploadable, ok := cache.uploadables.Load(id)
	if !ok {
		fail(writer, request, http.StatusNotFound, "GHA cache failed to find an uploadable "+
			"with ID %d", id)

		return
	}
	defer cache.uploadables.Delete(id)

	var jsonReq struct {
		Size int64 `json:"size"`
	}

	if err := render.DecodeJSON(request.Body, &jsonReq); err != nil {
		fail(writer, request, http.StatusBadRequest, "GHA cache failed to read/decode "+
			"the JSON passed to the commit uploadable endpoint: %v", err)

		return
	}

	finalizedUploadableReader, finalizedUploadableSize, err := uploadable.Finalize()
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed to "+
			"finalize uploadable %d: %v", id, err)

		return
	}

	if jsonReq.Size != finalizedUploadableSize {
		fail(writer, request, http.StatusBadRequest, "GHA cache detected a cache entry "+
			"size mismatch for uploadable %d: expected %d bytes, got %d bytes",
			id, finalizedUploadableSize, jsonReq.Size)

		return
	}

	req, err := http.NewRequestWithContext(request.Context(), http.MethodPost, cache.httpCacheURL(uploadable.Key,
		uploadable.Version), finalizedUploadableReader)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed to "+
			"upload the uploadable with ID %d: %v", id, err)

		return
	}

	// Explicitly set the Content-Length header,
	// otherwise HTTP 411 Length Required from
	// the upstream S3-compatible server
	req.ContentLength = finalizedUploadableSize

	// Set the Content-Type header to indicate
	// that we're sending arbitrary binary data
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed to "+
			"upload the uploadable with ID %d: %v", id, err)

		return
	}

	if !(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated) {
		fail(writer, request, resp.StatusCode, "GHA cache failed to "+
			"upload the uploadable with ID %d: got HTTP %d", id, resp.StatusCode)

		return
	}

	writer.WriteHeader(http.StatusCreated)
}

func (cache *GHACache) httpCacheURL(key string, version string) string {
	return fmt.Sprintf("http://%s/%s-%s", cache.cacheHost, url.PathEscape(key), url.PathEscape(version))
}

func getID(request *http.Request) (int64, bool) {
	idRaw := request.PathValue("id")

	id, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil {
		return 0, false
	}

	return id, true
}

func fail(writer http.ResponseWriter, request *http.Request, status int, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)

	log.Println(message)

	writer.WriteHeader(status)
	jsonResp := struct {
		Message string `json:"message"`
	}{
		Message: message,
	}
	render.JSON(writer, request, &jsonResp)
}
