package memory

import (
	"context"
	"errors"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/util"
	"io/ioutil"
	"os"
	"path"
	"syscall"
)

type Memory struct {
	fs billy.Filesystem
}

func New(fileContents map[string][]byte) (*Memory, error) {
	memory := &Memory{
		fs: memfs.New(),
	}

	for path, contents := range fileContents {
		if err := util.WriteFile(memory.fs, path, contents, 0600); err != nil {
			return nil, err
		}
	}

	return memory, nil
}

func (memory *Memory) Stat(ctx context.Context, path string) (*fs.FileInfo, error) {
	fileInfo, err := memory.fs.Stat(path)
	if err != nil {
		return nil, err
	}

	return &fs.FileInfo{IsDir: fileInfo.IsDir()}, nil
}

func (memory *Memory) Get(ctx context.Context, path string) ([]byte, error) {
	// Work around github.com/go-git/go-billy quirks
	// in regard to treatment of directories in memory FS
	fileInfo, err := memory.fs.Stat(path)
	if err == nil && fileInfo.IsDir() {
		return nil, fs.ErrNormalizedIsADirectory
	}

	file, err := memory.fs.Open(path)
	if err != nil {
		return nil, err
	}

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return fileBytes, nil
}

func (memory *Memory) ReadDir(ctx context.Context, path string) ([]string, error) {
	// Work around github.com/go-git/go-billy quirks
	// in regard to treatment of directories in memory FS
	fileInfo, err := memory.fs.Stat(path)
	if err == nil && !fileInfo.IsDir() {
		return nil, syscall.ENOTDIR
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	fileInfos, err := memory.fs.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, fileInfo := range fileInfos {
		result = append(result, fileInfo.Name())
	}

	return result, nil
}

func (memory *Memory) Join(elem ...string) string {
	return path.Join(elem...)
}
