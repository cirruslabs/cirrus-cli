package local

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Local struct {
	root string
}

func New(root string) *Local {
	return &Local{
		root: root,
	}
}

func (lfs *Local) Stat(ctx context.Context, path string) (*fs.FileInfo, error) {
	fileInfo, err := os.Stat(lfs.pivot(path))
	if err != nil {
		return nil, err
	}

	return &fs.FileInfo{IsDir: fileInfo.IsDir()}, nil
}

func (lfs *Local) Get(ctx context.Context, path string) ([]byte, error) {
	// To make Starlark scripts cross-platform, load statements are expected to always use slashes,
	// but to actually make this work on non-Unix platforms we need to adapt the path
	// to the current platform
	adaptedPath := filepath.FromSlash(path)

	return ioutil.ReadFile(lfs.pivot(adaptedPath))
}

func (lfs *Local) ReadDir(ctx context.Context, path string) ([]string, error) {
	fileInfos, err := ioutil.ReadDir(lfs.pivot(path))
	if err != nil {
		return nil, err
	}

	var result []string
	for _, fileInfo := range fileInfos {
		result = append(result, fileInfo.Name())
	}

	return result, nil
}

func (lfs *Local) pivot(path string) string {
	return filepath.Join(lfs.root, path)
}
