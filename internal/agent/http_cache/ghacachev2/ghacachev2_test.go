package ghacachev2_test

import (
	"bytes"
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache/cirruscimock"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/api/gharesults"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/twitchtv/twirp"
	"io"
	"net/http"
	"testing"
)

func TestGHACacheV2(t *testing.T) {
	testutil.NeedsContainerization(t)

	ctx := context.Background()

	client.InitClient(cirruscimock.ClientConn(t), "test", "test")

	httpCacheURL := "http://" + http_cache.Start()

	client := gharesults.NewCacheServiceJSONClient(httpCacheURL, &http.Client{})

	cacheKey := uuid.NewString()
	cacheValue := []byte("Hello, World!\n")

	// Ensure that an entry for our cache key is not present
	_, err := client.GetCacheEntryDownloadURL(ctx, &gharesults.GetCacheEntryDownloadURLRequest{
		Key: cacheKey,
	})
	var twirpError twirp.Error
	require.ErrorAs(t, err, &twirpError)
	require.Equal(t, twirp.NotFound, twirpError.Code())

	// Upload an entry for our cache key
	createCacheEntryRes, err := client.CreateCacheEntry(ctx, &gharesults.CreateCacheEntryRequest{
		Key: cacheKey,
	})
	require.NoError(t, err)
	require.True(t, createCacheEntryRes.Ok)

	uploadReq, err := http.NewRequest(http.MethodPut, createCacheEntryRes.SignedUploadUrl, bytes.NewReader(cacheValue))
	require.NoError(t, err)

	uploadResp, err := http.DefaultClient.Do(uploadReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, uploadResp.StatusCode)

	// Ensure that an entry for our cache key is present
	// and matches to what we've previously put in the cache
	getCacheEntryDownloadURLResp, err := client.GetCacheEntryDownloadURL(ctx, &gharesults.GetCacheEntryDownloadURLRequest{
		Key: cacheKey,
	})
	require.NoError(t, err)
	require.True(t, getCacheEntryDownloadURLResp.Ok)
	require.Equal(t, cacheKey, getCacheEntryDownloadURLResp.MatchedKey)

	downloadResp, err := http.Get(getCacheEntryDownloadURLResp.SignedDownloadUrl)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, downloadResp.StatusCode)

	downloadRespBodyBytes, err := io.ReadAll(downloadResp.Body)
	require.NoError(t, err)
	require.Equal(t, cacheValue, downloadRespBodyBytes)
}
