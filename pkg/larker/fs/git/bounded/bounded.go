package bounded

import (
	"errors"
	"fmt"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"os"
)

var ErrExhausted = errors.New("filesystem limit exhausted")

type FileSystem struct {
	maxFiles int
	maxBytes int

	filesCreated int
	bytesWritten int

	billy.Filesystem
}

type File struct {
	bfs *FileSystem

	billy.File
}

func NewFilesystem(maxBytes, maxFiles int) billy.Filesystem {
	return &FileSystem{
		maxFiles:   maxFiles,
		maxBytes:   maxBytes,
		Filesystem: memfs.New(),
	}
}

func (bfs *FileSystem) Create(filename string) (billy.File, error) {
	return bfs.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

func (bfs *FileSystem) Open(filename string) (billy.File, error) {
	return bfs.OpenFile(filename, os.O_RDONLY, 0)
}

func (bfs *FileSystem) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	if err := bfs.moreFiles(1); err != nil {
		return nil, err
	}

	file, err := bfs.Filesystem.OpenFile(filename, flag, perm)

	// Accounting
	if err == nil && (flag&os.O_CREATE) != 0 {
		bfs.filesCreated++
	}

	return &File{
		bfs:  bfs,
		File: file,
	}, err
}

func (bfs *FileSystem) MkdirAll(filename string, perm os.FileMode) error {
	if err := bfs.moreFiles(1); err != nil {
		return err
	}

	err := bfs.Filesystem.MkdirAll(filename, perm)

	// Accounting
	if err == nil {
		bfs.filesCreated++
	}

	return err
}

func (bfs *FileSystem) moreFiles(n int) error {
	if (bfs.filesCreated + n) > bfs.maxFiles {
		return fmt.Errorf("%w: attempted to create more than %d files", ErrExhausted, bfs.maxFiles)
	}

	return nil
}

func (bfs *FileSystem) moreBytes(n int) error {
	if (bfs.bytesWritten + n) > bfs.maxBytes {
		return fmt.Errorf("%w: attempted to write more than %d bytes", ErrExhausted, bfs.maxBytes)
	}

	return nil
}

func (bf *File) Write(p []byte) (int, error) {
	if err := bf.bfs.moreBytes(len(p)); err != nil {
		return -1, err
	}

	n, err := bf.File.Write(p)

	// Accounting
	if err == nil {
		bf.bfs.bytesWritten += n
	}

	return n, err
}
