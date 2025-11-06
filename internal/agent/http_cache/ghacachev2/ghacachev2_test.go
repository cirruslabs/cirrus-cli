package ghacachev2_test

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"io"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/azureblob"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache/cirruscimock"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/api/gharesults"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGHACacheV2(t *testing.T) {
	testutil.NeedsContainerization(t)

	ctx := context.Background()

	client.InitClient(cirruscimock.ClientConn(t), "test", "test")

	httpCacheURL := "http://" + http_cache.Start(ctx, http_cache.DefaultTransport(), http_cache.WithAzureBlobOpts(azureblob.WithUnexpectedEOFReader()))

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

func TestGHACacheV2UploadStream(t *testing.T) {
	testutil.NeedsContainerization(t)

	testCases := []struct {
		Name      string
		BlockSize int64
	}{
		{
			Name:      "normal-chunks",
			BlockSize: 5 * humanize.MiByte,
		},
		{
			Name:      "small-chunks",
			BlockSize: 1 * humanize.MiByte,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			client.InitClient(cirruscimock.ClientConn(t), "test", "test")

			httpCacheURL := "http://" + http_cache.Start(t.Context(), http_cache.DefaultTransport(), http_cache.WithAzureBlobOpts(azureblob.WithUnexpectedEOFReader()))

			client := gharesults.NewCacheServiceJSONClient(httpCacheURL, &http.Client{})

			cacheKey := uuid.NewString()
			cacheValue := make([]byte, 50*humanize.MiByte)
			_, err := cryptorand.Read(cacheValue)
			require.NoError(t, err)

			// Ensure that an entry for our cache key is not present
			getCacheEntryDownloadURLRes, err := client.GetCacheEntryDownloadURL(t.Context(),
				&gharesults.GetCacheEntryDownloadURLRequest{
					Key: cacheKey,
				},
			)
			require.NoError(t, err)
			require.False(t, getCacheEntryDownloadURLRes.Ok)

			// Upload an entry for our cache key
			createCacheEntryRes, err := client.CreateCacheEntry(t.Context(), &gharesults.CreateCacheEntryRequest{
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

			blockBlobClient, err := azblob.NewClientWithNoCredential(
				url.Scheme+"://"+url.Host+"/_azureblob",
				nil,
			)
			require.NoError(t, err)

			r := bytes.NewReader(cacheValue)

			_, err = blockBlobClient.UploadStream(t.Context(), url.ContainerName, url.BlobName, r,
				&azblob.UploadStreamOptions{
					BlockSize: testCase.BlockSize,
				},
			)
			require.NoError(t, err)

			// Ensure that an entry for our cache key is present
			// and matches to what we've previously put in the cache
			getCacheEntryDownloadURLResp, err := client.GetCacheEntryDownloadURL(t.Context(),
				&gharesults.GetCacheEntryDownloadURLRequest{
					Key: cacheKey,
				},
			)
			require.NoError(t, err)
			require.True(t, getCacheEntryDownloadURLResp.Ok)
			require.Equal(t, cacheKey, getCacheEntryDownloadURLResp.MatchedKey)

			downloadResp, err := http.Get(getCacheEntryDownloadURLResp.SignedDownloadUrl)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, downloadResp.StatusCode)

			downloadRespBodyBytes, err := io.ReadAll(downloadResp.Body)
			require.NoError(t, err)
			require.Equal(t, cacheValue, downloadRespBodyBytes)
		})
	}
}
