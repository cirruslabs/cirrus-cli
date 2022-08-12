package local

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	securejoin "github.com/cyphar/filepath-securejoin"
	"os"
	"path/filepath"
)

type Local struct {
	root string
	cwd  string
}

func New(root string) *Local {
	return &Local{
		root: root,
		cwd:  "/",
	}
}

func (lfs *Local) Chdir(path string) {
	lfs.cwd = path
}

func (lfs *Local) Stat(ctx context.Context, path string) (*fs.FileInfo, error) {
	pivotedPath, err := lfs.Pivot(path)
	if err != nil {
		return nil, err
	}

	fileInfo, err := os.Stat(pivotedPath)
	if err != nil {
		return nil, err
	}

	return &fs.FileInfo{IsDir: fileInfo.IsDir()}, nil
}

func (lfs *Local) Get(ctx context.Context, path string) ([]byte, error) {
	pivotedPath, err := lfs.Pivot(path)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(pivotedPath)
}

func (lfs *Local) ReadDir(ctx context.Context, path string) ([]string, error) {
	pivotedPath, err := lfs.Pivot(path)
	if err != nil {
		return nil, err
	}

	fileInfos, err := os.ReadDir(pivotedPath)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, fileInfo := range fileInfos {
		result = append(result, fileInfo.Name())
	}

	return result, nil
}

func (lfs *Local) Join(elem ...string) string {
	return filepath.Join(elem...)
}

func (lfs *Local) Pivot(path string) (string, error) {
	// To make Starlark scripts cross-platform, load statements are expected to always use slashes,
	// but to actually make this work on non-Unix platforms we need to adapt the path
	// to the current platform
	adaptedPath := filepath.FromSlash(path)

	// Pivot around current directory
	//
	// This doesn't need to be secure since as security
	// is already guaranteed by the SecureJoin below.
	cwdPath := filepath.Join(lfs.cwd, adaptedPath)

	// Pivot around root
	//
	// This needs to be secure to avoid lfs.root breakout.
	return securejoin.SecureJoin(lfs.root, cwdPath)
}
