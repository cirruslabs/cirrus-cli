package tuistcache

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	tuistapi "github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/tuistcache/api"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	ogenhttp "github.com/ogen-go/ogen/http"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"net/url"
	"time"
)

const APIMountPoint = "/tuistcache"

type NoSecurity struct{}

func (NoSecurity) HandleAuthorization(ctx context.Context, _ string, _ tuistapi.Authorization) (context.Context, error) {
	return ctx, nil
}

func (NoSecurity) HandleCookie(ctx context.Context, _ string, _ tuistapi.Cookie) (context.Context, error) {
	return ctx, nil
}

type TuistCache struct {
	server *tuistapi.Server

	tuistapi.UnimplementedHandler
}

func New() (*TuistCache, error) {
	tc := &TuistCache{}

	server, err := tuistapi.NewServer(tc, NoSecurity{})
	if err != nil {
		return nil, err
	}

	return &TuistCache{
		server: server,
	}, nil
}

func (tc *TuistCache) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	tc.server.ServeHTTP(writer, request)
}

func (tc *TuistCache) DownloadCacheArtifact(
	ctx context.Context,
	params tuistapi.DownloadCacheArtifactParams,
) (tuistapi.DownloadCacheArtifactRes, error) {
	cacheKey := getCacheKey(params.CacheCategory, params.ProjectID, params.Hash, params.Name)

	generateCacheUploadURLResponse, err := client.CirrusClient.GenerateCacheDownloadURLs(ctx, &api.CacheKey{
		TaskIdentification: client.CirrusTaskIdentification,
		CacheKey:           cacheKey,
	})
	if err != nil {
		if status, ok := status.FromError(err); ok && status.Code() == codes.NotFound {
			return &tuistapi.DownloadCacheArtifactNotFound{}, nil
		}

		return nil, err
	}

	if len(generateCacheUploadURLResponse.Urls) != 1 {
		return nil, ogenhttp.ErrInternalServerErrorResponse
	}

	return &tuistapi.CacheArtifactDownloadURL{
		Status: tuistapi.CacheArtifactDownloadURLStatusSuccess,
		Data: tuistapi.CacheArtifactDownloadURLData{
			URL:       generateCacheUploadURLResponse.Urls[0],
			ExpiresAt: int(time.Now().Add(time.Duration(10) * time.Minute).Unix()),
		},
	}, nil
}

func (tc *TuistCache) StartCacheArtifactMultipartUpload(
	ctx context.Context,
	params tuistapi.StartCacheArtifactMultipartUploadParams,
) (tuistapi.StartCacheArtifactMultipartUploadRes, error) {
	cacheKey := getCacheKey(params.CacheCategory, params.ProjectID, params.Hash, params.Name)

	multipartCacheUploadCreateResponse, err := client.CirrusClient.MultipartCacheUploadCreate(ctx, &api.CacheKey{
		TaskIdentification: client.CirrusTaskIdentification,
		CacheKey:           cacheKey,
	})
	if err != nil {
		return nil, err
	}

	return &tuistapi.ArtifactUploadID{
		Status: tuistapi.ArtifactUploadIDStatusSuccess,
		Data: tuistapi.ArtifactUploadIDData{
			UploadID: multipartCacheUploadCreateResponse.UploadId,
		},
	}, nil
}

func (tc *TuistCache) GenerateCacheArtifactMultipartUploadURL(
	ctx context.Context,
	params tuistapi.GenerateCacheArtifactMultipartUploadURLParams,
) (tuistapi.GenerateCacheArtifactMultipartUploadURLRes, error) {
	cacheKey := getCacheKey(params.CacheCategory, params.ProjectID, params.Hash, params.Name)

	multipartCacheUploadPartResponse, err := client.CirrusClient.MultipartCacheUploadPart(ctx, &api.MultipartCacheUploadPartRequest{
		CacheKey: &api.CacheKey{
			TaskIdentification: client.CirrusTaskIdentification,
			CacheKey:           cacheKey,
		},
		UploadId:      params.UploadID,
		PartNumber:    uint32(params.PartNumber),
		ContentLength: uint64(params.ContentLength.Or(0)),
	})
	if err != nil {
		return nil, err
	}

	return &tuistapi.ArtifactMultipartUploadURL{
		Status: tuistapi.ArtifactMultipartUploadURLStatusSuccess,
		Data: tuistapi.ArtifactMultipartUploadURLData{
			URL: multipartCacheUploadPartResponse.Url,
		},
	}, nil
}

func (tc *TuistCache) CompleteCacheArtifactMultipartUpload(
	ctx context.Context,
	req tuistapi.OptCompleteCacheArtifactMultipartUploadReq,
	params tuistapi.CompleteCacheArtifactMultipartUploadParams,
) (tuistapi.CompleteCacheArtifactMultipartUploadRes, error) {
	cacheKey := getCacheKey(params.CacheCategory, params.ProjectID, params.Hash, params.Name)

	var parts []*api.MultipartCacheUploadCommitRequest_Part

	for _, part := range req.Value.Parts {
		parts = append(parts, &api.MultipartCacheUploadCommitRequest_Part{
			PartNumber: uint32(part.PartNumber.Value),
			Etag:       part.Etag.Value,
		})
	}

	_, err := client.CirrusClient.MultipartCacheUploadCommit(ctx, &api.MultipartCacheUploadCommitRequest{
		CacheKey: &api.CacheKey{
			TaskIdentification: client.CirrusTaskIdentification,
			CacheKey:           cacheKey,
		},
		UploadId: params.UploadID,
		Parts:    parts,
	})
	if err != nil {
		return nil, err
	}

	return &tuistapi.CompleteCacheArtifactMultipartUploadOK{}, nil
}

func URL(httpCacheHost string) string {
	tuistCacheURL := url.URL{
		Scheme: "http",
		Host:   httpCacheHost,
		Path:   APIMountPoint,
	}

	return tuistCacheURL.String()
}

func getCacheKey(cacheCategory tuistapi.OptCacheCategory, projectID string, hash string, name string) string {
	return fmt.Sprintf("%s-%s-%s-%s", string(cacheCategory.Or(tuistapi.CacheCategoryBuilds)),
		projectID, hash, name)
}
