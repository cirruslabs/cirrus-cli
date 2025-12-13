package ghacache

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache/httprange"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache/uploadable"
	agentstorage "github.com/cirruslabs/cirrus-cli/internal/agent/storage"
	"github.com/getsentry/sentry-go"
	"github.com/go-chi/render"
	"github.com/puzpuzpuz/xsync/v3"
	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
	"math"
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
	backend     agentstorage.CacheBackend
	mux         *http.ServeMux
	uploadables *xsync.MapOf[int64, *uploadable.Uploadable]
}

func New(cacheHost string, backend agentstorage.CacheBackend) *GHACache {
	cache := &GHACache{
		cacheHost:   cacheHost,
		backend:     backend,
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

	keysWithVersions := lo.Map(keys, func(key string, _ int) string {
		return httpCacheKey(key, version)
	})

	cacheKeyPrefixes := keysWithVersions[1:]
	info, err := cache.backend.CacheInfo(request.Context(), keysWithVersions[0], cacheKeyPrefixes)
	if err != nil {
		if status, ok := status.FromError(err); ok && status.Code() == codes.NotFound {
			writer.WriteHeader(http.StatusNoContent)

			return
		}

		fail(writer, request, http.StatusInternalServerError, "GHA cache failed to "+
			"retrieve information about cache entry", "key", keys[0], "err", err)

		return
	}

	jsonResp := struct {
		Key string `json:"cacheKey"`
		URL string `json:"archiveLocation"`
	}{
		Key: strings.TrimPrefix(info.Key, httpCacheKey("", version)),
		URL: cache.httpCacheURL(info.Key),
	}

	render.JSON(writer, request, &jsonResp)
}

func (cache *GHACache) reserveUploadable(writer http.ResponseWriter, request *http.Request) {
	var jsonReq struct {
		Key     string `json:"key"`
		Version string `json:"version"`
	}

	if err := render.DecodeJSON(request.Body, &jsonReq); err != nil {
		fail(writer, request, http.StatusBadRequest, "GHA cache failed to read/decode the "+
			"JSON passed to the reserve uploadable endpoint", "err", err)

		return
	}

	jsonResp := struct {
		CacheID int64 `json:"cacheId"`
	}{
		CacheID: rand.Int63n(jsNumberMaxSafeInteger),
	}

	uploadID, err := cache.backend.CreateMultipartUpload(request.Context(), httpCacheKey(jsonReq.Key, jsonReq.Version), nil)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed to create "+
			"multipart upload", "key", jsonReq.Key, "version", jsonReq.Version, "err", err)

		return
	}

	cache.uploadables.Store(jsonResp.CacheID, uploadable.New(jsonReq.Key, jsonReq.Version, uploadID))

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
		fail(writer, request, http.StatusNotFound, "GHA cache failed to find an uploadable",
			"id", id)

		return
	}

	httpRanges, err := httprange.ParseRange(request.Header.Get("Content-Range"), math.MaxInt64)
	if err != nil {
		fail(writer, request, http.StatusBadRequest, "GHA cache failed to parse Content-Range header",
			"header_value", request.Header.Get("Content-Range"), "err", err)

		return
	}

	if len(httpRanges) != 1 {
		fail(writer, request, http.StatusBadRequest, "GHA cache expected exactly one Content-Range value",
			"expected", 1, "actual", len(httpRanges))

		return
	}

	partNumber, err := uploadable.RangeToPart.Tell(request.Context(), httpRanges[0].Start, httpRanges[0].Length)
	if err != nil {
		fail(writer, request, http.StatusBadRequest, "GHA cache failed to tell the part number for "+
			"Content-Range header", "header_value", request.Header.Get("Content-Range"), "err", err)

		return
	}

	urlInfo, err := cache.backend.UploadPartURL(request.Context(),
		httpCacheKey(uploadable.Key(), uploadable.Version()),
		uploadable.UploadID(),
		uint32(partNumber),
		uint64(httpRanges[0].Length),
	)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed create pre-signed "+
			"upload part URL", "key", uploadable.Key(), "version", uploadable.Version(),
			"part_number", partNumber, "err", err)

		return
	}

	uploadPartRequest, err := http.NewRequest(http.MethodPut, urlInfo.URL, request.Body)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed to create upload part "+
			"request", "key", uploadable.Key(), "version", uploadable.Version(), "part_number", partNumber,
			"err", err)

		return
	}

	// Golang's HTTP client won't send the Content-Length
	// because it doesn't know the request.Body's size,
	// so we do it for it, otherwise we'll get HTTP 403
	// from S3 since non-existent "Content-Length" header
	// is not equal to "Content-Length: X" that was pre-signed.
	uploadPartRequest.ContentLength = httpRanges[0].Length

	// Add headers mandated by the pre-signed request
	for key, value := range urlInfo.ExtraHeaders {
		uploadPartRequest.Header.Set(key, value)
	}

	uploadPartResponse, err := http.DefaultClient.Do(uploadPartRequest)
	if err != nil {
		// Return HTTP 502 to cause the cache-related code in the Actions Toolkit to make a re-try[1].
		//
		// [1]: https://github.com/actions/toolkit/blob/6dd369c0e648ed58d0ead326cf2426906ea86401/packages/cache/src/internal/requestUtils.ts#L24-L34
		fail(writer, request, http.StatusBadGateway, "GHA cache failed to upload part",
			"key", uploadable.Key(), "version", uploadable.Version(), "part_number", partNumber,
			"err", err)

		return
	}

	if uploadPartResponse.StatusCode != http.StatusOK {
		// We pass through the status code here so that the cache-related
		// code in the Actions Toolkit will hopefully make a re-try[1].
		//
		// [1]: https://github.com/actions/toolkit/blob/6dd369c0e648ed58d0ead326cf2426906ea86401/packages/cache/src/internal/requestUtils.ts#L24-L34
		fail(writer, request, uploadPartResponse.StatusCode, "GHA cache failed to upload part",
			"key", uploadable.Key(), "version", uploadable.Version(), "part_number", partNumber,
			"unexpected_status_code", uploadPartResponse.StatusCode)

		return
	}

	err = uploadable.AppendPart(uint32(partNumber), uploadPartResponse.Header.Get("ETag"), httpRanges[0].Length)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed to append part",
			"key", uploadable.Key(), "version", uploadable.Version(), "part_number", partNumber,
			"err", err)

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
		fail(writer, request, http.StatusNotFound, "GHA cache failed to find an uploadable",
			"id", id)

		return
	}

	var jsonReq struct {
		Size int64 `json:"size"`
	}

	if err := render.DecodeJSON(request.Body, &jsonReq); err != nil {
		fail(writer, request, http.StatusBadRequest, "GHA cache failed to read/decode "+
			"the JSON passed to the commit uploadable endpoint", "err", err)

		return
	}

	parts, partsSize, err := uploadable.Finalize()
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed to "+
			"finalize uploadable", "id", id, "err", err)

		return
	}

	if jsonReq.Size != partsSize {
		fail(writer, request, http.StatusBadRequest, "GHA cache detected a cache entry "+
			"size mismatch for uploadable", "id", id, "expected_bytes", partsSize,
			"actual_bytes", jsonReq.Size)

		return
	}

	err = cache.backend.CommitMultipartUpload(request.Context(),
		httpCacheKey(uploadable.Key(), uploadable.Version()),
		uploadable.UploadID(),
		parts,
	)
	if err != nil {
		fail(writer, request, http.StatusInternalServerError, "GHA cache failed to commit multipart upload",
			"id", uploadable.UploadID(), "key", uploadable.Key(), "version", uploadable.Version(),
			"err", err)

		return
	}

	// Delete uploadable only on success. GHA Runner can retry in case of a failure.
	cache.uploadables.Delete(id)

	writer.WriteHeader(http.StatusCreated)
}

func httpCacheKey(key string, version string) string {
	return fmt.Sprintf("%s-%s", url.PathEscape(version), url.PathEscape(key))
}

func (cache *GHACache) httpCacheURL(keyWithVersion string) string {
	return fmt.Sprintf("http://%s/%s", cache.cacheHost, url.PathEscape(keyWithVersion))
}

func getID(request *http.Request) (int64, bool) {
	idRaw := request.PathValue("id")

	id, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil {
		return 0, false
	}

	return id, true
}

func fail(writer http.ResponseWriter, request *http.Request, status int, msg string, args ...any) {
	// Report failure to the Sentry
	hub := sentry.GetHubFromContext(request.Context())

	hub.WithScope(func(scope *sentry.Scope) {
		scope.AddEventProcessor(func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Swap the exception type and value to work around
			// https://github.com/getsentry/sentry/issues/17837
			savedType := event.Exception[0].Type
			event.Exception[0].Type = event.Exception[0].Value
			event.Exception[0].Value = savedType

			return event
		})

		argsAsSentryContext := sentry.Context{}

		for _, chunk := range lo.Chunk(args, 2) {
			key := fmt.Sprintf("%v", chunk[0])

			var value string

			if len(chunk) > 1 {
				value = fmt.Sprintf("%v", chunk[1])
			}

			argsAsSentryContext[key] = value
		}

		scope.SetContext("Arguments", argsAsSentryContext)

		hub.CaptureException(errors.New(msg))
	})

	// Format failure message for non-structured consumers
	var stringBuilder strings.Builder
	logger := slog.New(slog.NewTextHandler(&stringBuilder, nil))
	logger.Error(msg, args...)
	message := stringBuilder.String()

	// Report failure to the logger
	slog.Error(msg, args...)

	// Report failure to the caller
	writer.WriteHeader(status)
	jsonResp := struct {
		Message string `json:"message"`
	}{
		Message: message,
	}
	render.JSON(writer, request, &jsonResp)
}
