package fs

import (
	"os"
)

type Dummy struct{}

func NewDummyFileSystem() *Dummy {
	return &Dummy{}
}

func (dfs *Dummy) Get(path string) ([]byte, error) {
	return nil, os.ErrNotExist
}
