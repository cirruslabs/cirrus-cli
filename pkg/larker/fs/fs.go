package fs

import (
	"context"
)

type FileSystem interface {
	Stat(ctx context.Context, path string) (*FileInfo, error)
	Get(ctx context.Context, path string) ([]byte, error)
	ReadDir(ctx context.Context, path string) ([]string, error)
	Join(elem ...string) string
}

type FileInfo struct {
	IsDir bool
}
