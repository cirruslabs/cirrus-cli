package failing

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
)

type Failing struct {
	err error
}

func New(err error) *Failing {
	return &Failing{
		err: err,
	}
}

func (ffs *Failing) Stat(ctx context.Context, path string) (*fs.FileInfo, error) {
	return nil, ffs.err
}

func (ffs *Failing) Get(ctx context.Context, path string) ([]byte, error) {
	return nil, ffs.err
}

func (ffs *Failing) ReadDir(ctx context.Context, path string) ([]string, error) {
	return nil, ffs.err
}
