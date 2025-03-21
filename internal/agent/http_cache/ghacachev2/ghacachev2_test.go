package ghacachev2_test

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache/cirruscimock"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/api/gharesults"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

func TestGHACacheV2(t *testing.T) {
	testutil.NeedsContainerization(t)

	ctx := context.Background()

	client.InitClient(cirruscimock.ClientConn(t), "test", "test")

	httpCacheURL := "http://" + http_cache.Start(http_cache.DefaultTransport())

	client := gharesults.NewCacheServiceJSONClient(httpCacheURL, &http.Client{})

	cacheKey := uuid.NewString()
	cacheValue := []byte("Hello, World!\n")

	// Ensure that an entry for our cache key is not present
	getCacheEntryDownloadURLRes, err := client.GetCacheEntryDownloadURL(ctx, &gharesults.GetCacheEntryDownloadURLRequest{
		Key: cacheKey,
	})
	require.NoError(t, err)
	require.False(t, getCacheEntryDownloadURLRes.Ok)

	// Upload an entry for our cache key
	createCacheEntryRes, err := client.CreateCacheEntry(ctx, &gharesults.CreateCacheEntryRequest{
		Key: cacheKey,
	})
	require.NoError(t, err)
	require.True(t, createCacheEntryRes.Ok)

	// Feed the returned pre-signed upload URL to Azure Blob client
	blockBlobClient, err := azblob.NewBlockBlobClientWithNoCredential(createCacheEntryRes.SignedUploadUrl,
		nil)
	require.NoError(t, err)

	_, err = blockBlobClient.UploadBuffer(ctx, cacheValue, azblob.UploadOption{})
	require.NoError(t, err)

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

	// Ensure that blob properties can be retrieved,
	// this is actively used by GitHub Actions Toolkit
	// to determine whether to enable parallel download
	// or not.
	properties, err := blockBlobClient.GetProperties(ctx, &azblob.BlobGetPropertiesOptions{})
	require.NoError(t, err)
	require.NotNil(t, properties.ContentLength)
	require.EqualValues(t, len(cacheValue), *properties.ContentLength)

	// Ensure that HTTP range requests are supported
	buf := make([]byte, 4)
	err = blockBlobClient.DownloadToBuffer(ctx, 10, 4, buf, azblob.DownloadOptions{})
	require.NoError(t, err)
}
