package http_cache

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/tuistcache"
	"net/http"
)

type Option func(mux *http.ServeMux)

func WithTuistCache(tuistCache *tuistcache.TuistCache) Option {
	return func(mux *http.ServeMux) {
		mux.Handle(tuistcache.APIMountPoint+"/", http.StripPrefix(tuistcache.APIMountPoint, tuistCache))
	}
}
