package http_cache

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	omnicacheserver "github.com/cirruslabs/omni-cache/pkg/server"
	omnistorage "github.com/cirruslabs/omni-cache/pkg/storage"
)

const defaultCacheAddress = "127.0.0.1:12321"

func Start(ctx context.Context, backend omnistorage.BlobStorageBackend) string {
	if backend == nil {
		panic("http_cache.Start: backend is required")
	}

	ensureSocketHome()

	srv, err := omnicacheserver.StartDefault(ctx, backend)
	if err != nil {
		slog.Error("Failed to start http cache server", "address", defaultCacheAddress, "err", err)
		return defaultCacheAddress
	}

	if srv.Addr != "" {
		slog.Info("Starting http cache server", "address", srv.Addr)
		return srv.Addr
	}

	slog.Info("Starting http cache server", "address", defaultCacheAddress)
	return defaultCacheAddress
}

func ensureSocketHome() {
	homeDir, err := os.UserHomeDir()
	if err == nil && homeDir != "" {
		socketDir := filepath.Join(homeDir, ".cirruslabs")
		if err := os.MkdirAll(socketDir, 0o700); err == nil {
			return
		}
		slog.Warn("Failed to create omni-cache socket dir, falling back to temp dir", "path", socketDir, "err", err)
	} else if err != nil {
		slog.Warn("Failed to resolve home dir for omni-cache socket, falling back to temp dir", "err", err)
	}

	tempDir := os.TempDir()
	if tempDir == "" {
		return
	}
	if err := os.Setenv("HOME", tempDir); err != nil {
		slog.Warn("Failed to set HOME for omni-cache socket path", "err", err)
		return
	}
	_ = os.MkdirAll(filepath.Join(tempDir, ".cirruslabs"), 0o700)
}
