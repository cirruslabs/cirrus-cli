package http_cache

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log/slog"
	"net/http"
)

func (httpCache *HTTPCache) downloadCacheViaRPC(w http.ResponseWriter, r *http.Request, cacheKey string) {
	cacheStream, err := client.CirrusClient.DownloadCache(r.Context(), &api.DownloadCacheRequest{
		TaskIdentification: client.CirrusTaskIdentification,
		CacheKey:           cacheKey,
	})
	if err != nil {
		slog.Error("Cache download initialization via RPC failed", "cache_key", cacheKey, "err", err)

		if status.Code(err) == codes.NotFound {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}

		return
	}

	for {
		chunk, err := cacheStream.Recv()
		if err != nil {
			if err == io.EOF {
				slog.Info("Cache download via RPC finished", "cache_key", cacheKey)
			} else {
				slog.Error("Cache download via RPC failed", "cache_key", cacheKey, "err", err)

				if status.Code(err) == codes.NotFound {
					w.WriteHeader(http.StatusNotFound)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
				}
			}

			return
		}

		if chunk.RedirectUrl != "" {
			slog.Info("Cache download via RPC requested redirect", "cache_key", cacheKey)
			httpCache.proxyDownloadFromURLs(w, r, cacheKey, []string{chunk.RedirectUrl})

			return
		}

		if len(chunk.Data) == 0 {
			continue
		}

		if _, err := w.Write(chunk.Data); err != nil {
			slog.Error("Cache download via RPC failed", "cache_key", cacheKey, "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}
	}
}

func uploadCacheEntryViaRPC(w http.ResponseWriter, r *http.Request, cacheKey string) {
	uploadCacheClient, err := client.CirrusClient.UploadCache(r.Context())
	if err != nil {
		slog.Error("Cache upload initialization via RPC failed", "cache_key", cacheKey, "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if err := uploadCacheClient.Send(&api.CacheEntry{
		Value: &api.CacheEntry_Key{
			Key: &api.CacheKey{
				TaskIdentification: client.CirrusTaskIdentification,
				CacheKey:           cacheKey,
			},
		},
	}); err != nil {
		slog.Error("Cache upload via RPC failed", "cache_key", cacheKey, "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	buf := make([]byte, 1024*1024)

	for {
		n, err := r.Body.Read(buf)
		if n != 0 {
			err = uploadCacheClient.Send(&api.CacheEntry{
				Value: &api.CacheEntry_Chunk{
					Chunk: &api.DataChunk{
						Data: buf[:n],
					},
				},
			})
			if err != nil {
				slog.Error("Cache upload via RPC failed", "cache_key", cacheKey, "err", err)
				w.WriteHeader(http.StatusInternalServerError)

				_, _ = uploadCacheClient.CloseAndRecv()

				return
			}
		}
		if err == io.EOF {
			slog.Info("Cache upload via RPC finished", "cache_key", cacheKey)

			break
		}
		if err != nil {
			slog.Error("Cache upload via RPC failed", "cache_key", cacheKey, "err", err)
			w.WriteHeader(http.StatusBadRequest)

			_, _ = uploadCacheClient.CloseAndRecv()

			return
		}
	}

	if _, err := uploadCacheClient.CloseAndRecv(); err != nil {
		slog.Error("Cache upload via RPC failed", "cache_key", cacheKey, "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
}
