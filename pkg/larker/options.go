package larker

import (
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
)

type Option func(*Larker)

func WithFileSystem(fs fs.FileSystem) Option {
	return func(e *Larker) {
		e.fs = fs
	}
}
