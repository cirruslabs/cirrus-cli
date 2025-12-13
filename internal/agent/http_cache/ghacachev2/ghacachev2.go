package ghacachev2

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/azureblob"
	agentstorage "github.com/cirruslabs/cirrus-cli/internal/agent/storage"
	"github.com/cirruslabs/cirrus-cli/pkg/api/gharesults"
	"github.com/samber/lo"
	"github.com/twitchtv/twirp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"hash/fnv"
	"net/http"
	"net/url"
	"strings"
)

// Interface guard
//
// Ensures that Cache struct implements gharesults.CacheService interface.
var _ gharesults.CacheService = (*Cache)(nil)

const APIMountPoint = "/twirp"

type Cache struct {
	cacheHost   string
	backend     agentstorage.CacheBackend
	twirpServer gharesults.TwirpServer
}

func New(cacheHost string, backend agentstorage.CacheBackend) *Cache {
	if backend == nil {
		panic("ghacachev2.New: backend is required")
	}
	cache := &Cache{
		cacheHost: cacheHost,
		backend:   backend,
	}

	cache.twirpServer = gharesults.NewCacheServiceServer(cache)

	return cache
}

func (cache *Cache) PathPrefix() string {
	return cache.twirpServer.PathPrefix()
}

func (cache *Cache) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	cache.twirpServer.ServeHTTP(writer, request)
}

func (cache *Cache) GetCacheEntryDownloadURL(ctx context.Context, request *gharesults.GetCacheEntryDownloadURLRequest) (*gharesults.GetCacheEntryDownloadURLResponse, error) {
	cacheKeyPrefixes := lo.Map(request.RestoreKeys, func(restoreKey string, _ int) string {
		return httpCacheKey(restoreKey, request.Version)
	})
	info, err := cache.backend.CacheInfo(ctx, httpCacheKey(request.Key, request.Version), cacheKeyPrefixes)
	if err != nil {
		if status, ok := status.FromError(err); ok && status.Code() == codes.NotFound {
			return &gharesults.GetCacheEntryDownloadURLResponse{
				Ok: false,
			}, nil
		}

		return nil, twirp.NewErrorf(twirp.Internal, "GHA cache v2 failed to retrieve information "+
			"about cache entry with key %q and version %q: %v", request.Key, request.Version, err)
	}

	return &gharesults.GetCacheEntryDownloadURLResponse{
		Ok:                true,
		SignedDownloadUrl: cache.azureBlobURL(info.Key),
		MatchedKey:        strings.TrimPrefix(info.Key, httpCacheKey("", request.Version)),
	}, nil
}

func (cache *Cache) CreateCacheEntry(ctx context.Context, request *gharesults.CreateCacheEntryRequest) (*gharesults.CreateCacheEntryResponse, error) {
	return &gharesults.CreateCacheEntryResponse{
		Ok:              true,
		SignedUploadUrl: cache.azureBlobURL(httpCacheKey(request.Key, request.Version)),
	}, nil
}

func (cache *Cache) FinalizeCacheEntryUpload(ctx context.Context, request *gharesults.FinalizeCacheEntryUploadRequest) (*gharesults.FinalizeCacheEntryUploadResponse, error) {
	hash := fnv.New64a()

	_, _ = hash.Write([]byte(request.Key))
	_, _ = hash.Write([]byte(fmt.Sprintf("%d", request.SizeBytes)))
	_, _ = hash.Write([]byte(request.Version))

	return &gharesults.FinalizeCacheEntryUploadResponse{
		Ok:      true,
		EntryId: int64(hash.Sum64()),
	}, nil
}

func httpCacheKey(key string, version string) string {
	return fmt.Sprintf("%s-%s", version, key)
}

func (cache *Cache) azureBlobURL(keyWithVersion string) string {
	return fmt.Sprintf("http://%s%s/%s", cache.cacheHost, azureblob.APIMountPoint,
		url.PathEscape(keyWithVersion))
}
