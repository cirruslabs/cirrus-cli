package dummy

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"os"
)

type Dummy struct{}

func New() *Dummy {
	return &Dummy{}
}

func (dfs *Dummy) Stat(ctx context.Context, path string) (fs.FileInfo, error) {
	return nil, os.ErrNotExist
}

func (dfs *Dummy) Get(ctx context.Context, path string) ([]byte, error) {
	return nil, os.ErrNotExist
}

func (dfs *Dummy) ReadDir(ctx context.Context, path string) ([]string, error) {
	return nil, os.ErrNotExist
}
