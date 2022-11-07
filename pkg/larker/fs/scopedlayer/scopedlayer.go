package scopedlayer

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	pathpkg "path"
)

type ScopedLayer struct {
	fs           fs.FileSystem
	scopedAtPath string
}

func New(fs fs.FileSystem, scopedAtPath string) *ScopedLayer {
	return &ScopedLayer{
		fs:           fs,
		scopedAtPath: scopedAtPath,
	}
}

func (sl *ScopedLayer) Stat(ctx context.Context, path string) (*fs.FileInfo, error) {
	return sl.fs.Stat(ctx, pathpkg.Join(sl.scopedAtPath, path))
}

func (sl *ScopedLayer) Get(ctx context.Context, path string) ([]byte, error) {
	return sl.fs.Get(ctx, pathpkg.Join(sl.scopedAtPath, path))
}

func (sl *ScopedLayer) ReadDir(ctx context.Context, path string) ([]string, error) {
	return sl.fs.ReadDir(ctx, pathpkg.Join(sl.scopedAtPath, path))
}

func (sl *ScopedLayer) Join(elem ...string) string {
	return sl.fs.Join(elem...)
}
