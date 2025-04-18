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

	httpCacheURL := "http://" + http_cache.Start(ctx, http_cache.DefaultTransport(), false)

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
	//
	// Unfortunately azblob.ParseURL() just drops the rest of the path,
	// and has no ServiceURL() convenience method, so we have to manually
	// add the "://" and "/_azureblob" below.
	url, err := azblob.ParseURL(createCacheEntryRes.SignedUploadUrl)
	require.NoError(t, err)

	blockBlobClient, err := azblob.NewClientWithNoCredential(url.Scheme+"://"+url.Host+"/_azureblob", nil)
	require.NoError(t, err)

	_, err = blockBlobClient.UploadBuffer(ctx, url.ContainerName, url.BlobName, cacheValue, &azblob.UploadBufferOptions{})
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
	resp, err := blockBlobClient.DownloadStream(ctx, url.ContainerName, url.BlobName, &azblob.DownloadStreamOptions{})
	require.NoError(t, err)
	require.NotNil(t, resp.ContentLength)
	require.EqualValues(t, len(cacheValue), *resp.ContentLength)

	// Ensure that HTTP range requests are supported
	buf := make([]byte, 5)
	n, err := blockBlobClient.DownloadBuffer(ctx, url.ContainerName, url.BlobName, buf, &azblob.DownloadBufferOptions{
		Range: azblob.HTTPRange{
			Offset: 7,
			Count:  5,
		},
	})
	require.NoError(t, err)
	require.EqualValues(t, 5, n)
	require.Equal(t, []byte("World"), buf)
}
