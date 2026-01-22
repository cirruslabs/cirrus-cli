package storage

import (
	"context"

	"github.com/cirruslabs/cirrus-cli/pkg/api"
	omnistorage "github.com/cirruslabs/omni-cache/pkg/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CirrusStoreBackend struct {
	omnistorage.MultipartBlobStorageBackend

	client             api.CirrusCIServiceClient
	taskIdentification *api.TaskIdentification
}

func NewCirrusStoreBackend(client api.CirrusCIServiceClient, taskIdentification *api.TaskIdentification) *CirrusStoreBackend {
	return &CirrusStoreBackend{
		client:             client,
		taskIdentification: taskIdentification,
	}
}

func (c *CirrusStoreBackend) DownloadURLs(ctx context.Context, key string) ([]*omnistorage.URLInfo, error) {
	response, err := c.client.GenerateCacheDownloadURLs(ctx, &api.CacheKey{
		TaskIdentification: c.taskIdentification,
		CacheKey:           key,
	})
	if err != nil {
		return nil, err
	}

	urls := make([]*omnistorage.URLInfo, 0, len(response.Urls))

	for _, url := range response.Urls {
		urls = append(urls, &omnistorage.URLInfo{URL: url})
	}

	return urls, nil
}

func (c *CirrusStoreBackend) UploadURL(ctx context.Context, key string, metadate map[string]string) (*omnistorage.URLInfo, error) {
	response, err := c.client.GenerateCacheUploadURL(ctx, &api.CacheKey{
		TaskIdentification: c.taskIdentification,
		CacheKey:           key,
	})
	if err != nil {
		return nil, err
	}

	return &omnistorage.URLInfo{
		URL:          response.Url,
		ExtraHeaders: response.ExtraHeaders,
	}, nil
}

func (c *CirrusStoreBackend) CreateMultipartUpload(ctx context.Context, key string, metadata map[string]string) (uploadID string, err error) {
	response, err := c.client.MultipartCacheUploadCreate(ctx, &api.CacheKey{
		TaskIdentification: c.taskIdentification,
		CacheKey:           key,
	})
	if err != nil {
		return "", err
	}

	return response.UploadId, nil
}

func (c *CirrusStoreBackend) UploadPartURL(ctx context.Context, key string, uploadID string, partNumber uint32, contentLength uint64) (*omnistorage.URLInfo, error) {
	response, err := c.client.MultipartCacheUploadPart(ctx, &api.MultipartCacheUploadPartRequest{
		CacheKey: &api.CacheKey{
			TaskIdentification: c.taskIdentification,
			CacheKey:           key,
		},
		UploadId:      uploadID,
		PartNumber:    partNumber,
		ContentLength: contentLength,
	})
	if err != nil {
		return nil, err
	}

	return &omnistorage.URLInfo{
		URL:          response.Url,
		ExtraHeaders: response.ExtraHeaders,
	}, nil
}

func (c *CirrusStoreBackend) CommitMultipartUpload(ctx context.Context, key string, uploadID string, parts []omnistorage.MultipartUploadPart) error {
	apiParts := make([]*api.MultipartCacheUploadCommitRequest_Part, 0, len(parts))

	for _, part := range parts {
		apiParts = append(apiParts, &api.MultipartCacheUploadCommitRequest_Part{
			PartNumber: part.PartNumber,
			Etag:       part.ETag,
		})
	}

	_, err := c.client.MultipartCacheUploadCommit(ctx, &api.MultipartCacheUploadCommitRequest{
		CacheKey: &api.CacheKey{
			TaskIdentification: c.taskIdentification,
			CacheKey:           key,
		},
		UploadId: uploadID,
		Parts:    apiParts,
	})

	return err
}

func (c *CirrusStoreBackend) CacheInfo(ctx context.Context, key string, prefixes []string) (*omnistorage.CacheInfo, error) {
	response, err := c.client.CacheInfo(ctx, &api.CacheInfoRequest{
		TaskIdentification: c.taskIdentification,
		CacheKey:           key,
		CacheKeyPrefixes:   prefixes,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, omnistorage.ErrCacheNotFound
		}
		return nil, err
	}

	info := response.GetInfo()
	if info == nil {
		return nil, omnistorage.ErrCacheNotFound
	}

	return &omnistorage.CacheInfo{
		Key:       info.Key,
		SizeBytes: info.SizeInBytes,
	}, nil
}

func (c *CirrusStoreBackend) DeleteCache(ctx context.Context, key string) error {
	_, err := c.client.DeleteCache(ctx, &api.DeleteCacheRequest{
		TaskIdentification: c.taskIdentification,
		CacheKey:           key,
	})

	return err
}
