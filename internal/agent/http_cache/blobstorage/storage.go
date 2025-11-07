package blobstorage

import (
	"context"
)

type URLInfo struct {
	URL          string
	ExtraHeaders map[string]string
}

type MultipartPart struct {
	PartNumber uint32
	ETag       string
}

type CacheInfo struct {
	Key                string
	SizeInBytes        int64
	CreatedByTaskID    string
	OldCreatedByTaskID int64
}

type BlobStorageBacked interface {
	DownloadURLs(ctx context.Context, key string) ([]*URLInfo, error)
	UploadURL(ctx context.Context, key string, metadate map[string]string) (*URLInfo, error)
	MultipartUploadCreate(ctx context.Context, key string) (string, error)
	MultipartUploadPartURL(ctx context.Context, key string, uploadID string, partNumber uint32, contentLength uint64) (*URLInfo, error)
	MultipartUploadCommit(ctx context.Context, key string, uploadID string, parts []*MultipartPart) error
	Info(ctx context.Context, key string, prefixes []string) (*CacheInfo, error)
	Delete(ctx context.Context, key string) error
}
