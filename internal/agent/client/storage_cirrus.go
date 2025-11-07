package client

import (
	"context"

	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/blobstorage"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
)

type cirrusBlobStorage struct {
	client api.CirrusCIServiceClient
}

func NewCirrusBlobStorage(client api.CirrusCIServiceClient) blobstorage.BlobStorageBacked {
	return &cirrusBlobStorage{client: client}
}

func (s *cirrusBlobStorage) Info(ctx context.Context, key string, prefixes []string) (*blobstorage.CacheInfo, error) {
	request := &api.CacheInfoRequest{
		TaskIdentification: CirrusTaskIdentification,
		CacheKey:           key,
		CacheKeyPrefixes:   prefixes,
	}

	response, err := s.client.CacheInfo(ctx, request)
	if err != nil {
		return nil, err
	}

	return &blobstorage.CacheInfo{
		Key:                response.Info.GetKey(),
		SizeInBytes:        response.Info.GetSizeInBytes(),
		CreatedByTaskID:    response.Info.GetCreatedByTaskId(),
		OldCreatedByTaskID: response.Info.GetOldCreatedByTaskId(),
	}, nil
}

func (s *cirrusBlobStorage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteCache(ctx, &api.DeleteCacheRequest{
		TaskIdentification: CirrusTaskIdentification,
		CacheKey:           key,
	})

	return err
}

func (s *cirrusBlobStorage) UploadURL(ctx context.Context, key string, metadata map[string]string) (*blobstorage.URLInfo, error) {
	resp, err := s.client.GenerateCacheUploadURL(ctx, s.cacheKey(key))
	if err != nil {
		return nil, err
	}

	return &blobstorage.URLInfo{
		URL:          resp.GetUrl(),
		ExtraHeaders: resp.GetExtraHeaders(),
	}, nil
}

func (s *cirrusBlobStorage) DownloadURLs(ctx context.Context, key string) ([]*blobstorage.URLInfo, error) {
	resp, err := s.client.GenerateCacheDownloadURLs(ctx, s.cacheKey(key))
	if err != nil {
		return nil, err
	}

	urlInfos := make([]*blobstorage.URLInfo, len(resp.GetUrls()))
	for i, url := range resp.GetUrls() {
		urlInfos[i] = &blobstorage.URLInfo{URL: url}
	}

	return urlInfos, nil
}

func (s *cirrusBlobStorage) MultipartUploadCreate(ctx context.Context, key string) (string, error) {
	resp, err := s.client.MultipartCacheUploadCreate(ctx, s.cacheKey(key))
	if err != nil {
		return "", err
	}

	return resp.GetUploadId(), nil
}

func (s *cirrusBlobStorage) MultipartUploadPartURL(ctx context.Context, key string, uploadID string, partNumber uint32, contentLength uint64) (*blobstorage.URLInfo, error) {
	resp, err := s.client.MultipartCacheUploadPart(ctx, &api.MultipartCacheUploadPartRequest{
		CacheKey:      s.cacheKey(key),
		UploadId:      uploadID,
		PartNumber:    partNumber,
		ContentLength: contentLength,
	})
	if err != nil {
		return nil, err
	}

	return &blobstorage.URLInfo{
		URL:          resp.GetUrl(),
		ExtraHeaders: resp.GetExtraHeaders(),
	}, nil
}

func (s *cirrusBlobStorage) MultipartUploadCommit(ctx context.Context, key string, uploadID string, parts []*blobstorage.MultipartPart) error {
	apiParts := make([]*api.MultipartCacheUploadCommitRequest_Part, 0, len(parts))
	for _, part := range parts {
		if part == nil {
			continue
		}

		apiParts = append(apiParts, &api.MultipartCacheUploadCommitRequest_Part{
			PartNumber: part.PartNumber,
			Etag:       part.ETag,
		})
	}

	_, err := s.client.MultipartCacheUploadCommit(ctx, &api.MultipartCacheUploadCommitRequest{
		CacheKey: s.cacheKey(key),
		UploadId: uploadID,
		Parts:    apiParts,
	})

	return err
}

func (s *cirrusBlobStorage) cacheKey(key string) *api.CacheKey {
	return &api.CacheKey{
		TaskIdentification: CirrusTaskIdentification,
		CacheKey:           key,
	}
}
