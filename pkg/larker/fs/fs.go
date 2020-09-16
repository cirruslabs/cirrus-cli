package fs

import "context"

type FileSystem interface {
	Get(ctx context.Context, path string) ([]byte, error)
}
