package http_cache

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/azureblob"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/blobstorage"
)

type Option func(httpCache *HTTPCache)

func WithAzureBlobOpts(opts ...azureblob.Option) Option {
	return func(httpCache *HTTPCache) {
		httpCache.azureBlobOpts = opts
	}
}

func WithBlobStorage(storage blobstorage.BlobStorageBacked) Option {
	return func(httpCache *HTTPCache) {
		httpCache.blobStorage = storage
	}
}
