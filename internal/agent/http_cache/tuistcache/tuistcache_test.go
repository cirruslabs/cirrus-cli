package tuistcache_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache/cirruscimock"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/tuistcache"
	tuistapi "github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/tuistcache/api"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestTuistCache(t *testing.T) {
	testutil.NeedsContainerization(t)

	ctx := context.Background()

	client.InitClient(cirruscimock.ClientConn(t), "test", "test")

	tuistCache, err := tuistcache.New()
	require.NoError(t, err)

	tuistCacheURL := tuistcache.URL(http_cache.Start(http_cache.DefaultTransport(),
		http_cache.WithTuistCache(tuistCache)))

	tuistCacheClient, err := tuistapi.NewClient(tuistCacheURL)
	require.NoError(t, err)

	projectID := "account-name/project-name"
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte("doesn't matter")))
	name := "test"

	// Ensure that cache entry does not exist
	downloadCacheArtifactResp, err := tuistCacheClient.DownloadCacheArtifact(ctx, tuistapi.DownloadCacheArtifactParams{
		ProjectID: projectID,
		Hash:      hash,
		Name:      name,
	})
	require.NoError(t, err)

	require.IsType(t, &tuistapi.CacheArtifactDownloadURL{}, downloadCacheArtifactResp)
	cacheArtifactDownloadUrl := downloadCacheArtifactResp.(*tuistapi.CacheArtifactDownloadURL)

	cacheArtifactDownloadUrlResp, err := http.Get(cacheArtifactDownloadUrl.Data.URL)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, cacheArtifactDownloadUrlResp.StatusCode)

	// Create the cache entry
	startCacheArtifactMultipartUploadResp, err := tuistCacheClient.StartCacheArtifactMultipartUpload(
		ctx,
		tuistapi.StartCacheArtifactMultipartUploadParams{
			ProjectID: projectID,
			Hash:      hash,
			Name:      name,
		},
	)
	require.NoError(t, err)
	require.IsType(t, &tuistapi.ArtifactUploadID{}, startCacheArtifactMultipartUploadResp)

	artifactUploadID := startCacheArtifactMultipartUploadResp.(*tuistapi.ArtifactUploadID)

	generateCacheArtifactMultipartUploadURLResp, err := tuistCacheClient.GenerateCacheArtifactMultipartUploadURL(
		ctx,
		tuistapi.GenerateCacheArtifactMultipartUploadURLParams{
			ProjectID: projectID,
			Hash:      hash,
			Name:      name,

			UploadID:   artifactUploadID.Data.UploadID,
			PartNumber: 1,
		},
	)
	require.NoError(t, err)
	require.IsType(t, &tuistapi.ArtifactMultipartUploadURL{}, generateCacheArtifactMultipartUploadURLResp)

	artifactMultipartUploadURL := generateCacheArtifactMultipartUploadURLResp.(*tuistapi.ArtifactMultipartUploadURL)

	req, err := http.NewRequest(http.MethodPut, artifactMultipartUploadURL.Data.URL, bytes.NewReader([]byte("Hello, World!")))
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	completeCacheArtifactMultipartUploadResp, err := tuistCacheClient.CompleteCacheArtifactMultipartUpload(
		ctx,
		tuistapi.NewOptCompleteCacheArtifactMultipartUploadReq(tuistapi.CompleteCacheArtifactMultipartUploadReq{
			Parts: []tuistapi.CompleteCacheArtifactMultipartUploadReqPartsItem{
				{
					PartNumber: tuistapi.NewOptInt(1),
					Etag:       tuistapi.NewOptString(resp.Header.Get("ETag")),
				},
			},
		}),
		tuistapi.CompleteCacheArtifactMultipartUploadParams{
			ProjectID: projectID,
			Hash:      hash,
			Name:      name,

			UploadID: artifactUploadID.Data.UploadID,
		},
	)
	require.NoError(t, err)
	require.IsType(t, &tuistapi.CompleteCacheArtifactMultipartUploadOK{}, completeCacheArtifactMultipartUploadResp)

	// Ensure that the cache entry now exists
	downloadCacheArtifactResp, err = tuistCacheClient.DownloadCacheArtifact(ctx, tuistapi.DownloadCacheArtifactParams{
		ProjectID: projectID,
		Hash:      hash,
		Name:      name,
	})
	require.NoError(t, err)
	require.IsType(t, &tuistapi.CacheArtifactDownloadURL{}, downloadCacheArtifactResp)
}
