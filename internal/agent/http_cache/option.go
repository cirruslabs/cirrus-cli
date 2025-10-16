package http_cache

import "github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/azureblob"

type Option func(httpCache *HTTPCache)

func WithAzureBlobOpts(opts ...azureblob.Option) Option {
	return func(httpCache *HTTPCache) {
		httpCache.azureBlobOpts = opts
	}
}
