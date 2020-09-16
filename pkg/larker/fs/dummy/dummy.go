package dummy

import (
	"context"
	"os"
)

type Dummy struct{}

func New() *Dummy {
	return &Dummy{}
}

func (dfs *Dummy) Get(ctx context.Context, path string) ([]byte, error) {
	return nil, os.ErrNotExist
}
