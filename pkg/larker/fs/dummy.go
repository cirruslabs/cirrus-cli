package fs

import (
	"context"
	"os"
)

type Dummy struct{}

func NewDummyFileSystem() *Dummy {
	return &Dummy{}
}

func (dfs *Dummy) Get(ctx context.Context, path string) ([]byte, error) {
	return nil, os.ErrNotExist
}
