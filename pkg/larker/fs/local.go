package fs

import (
	"io/ioutil"
	"path/filepath"
)

type Local struct {
	root string
}

func NewLocalFileSystem(root string) *Local {
	return &Local{
		root: root,
	}
}

func (lfs *Local) Get(path string) ([]byte, error) {
	// To make Starlark scripts cross-platform, load statements are expected to always use slashes,
	// but to actually make this work on non-Unix platforms we need to adapt the path
	// to the current platform
	adaptedPath := filepath.FromSlash(path)

	return ioutil.ReadFile(filepath.Join(lfs.root, adaptedPath))
}
