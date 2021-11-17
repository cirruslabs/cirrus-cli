package git

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/git/bounded"
	"github.com/docker/go-units"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"syscall"
)

var (
	ErrRetrievalFailed = errors.New("failed to retrieve a file from Git repository")
)

type Git struct {
	worktree *git.Worktree
}

func NewGit(ctx context.Context, url string, Revision string) (*Git, error) {
	const (
		cacheBytes = 1 * units.MiB

		storageBytes = 4 * units.MiB
		storageFiles = 4096

		filesystemBytes = 4 * units.MiB
		filesystemFiles = 4096
	)

	boundedCache := cache.NewObjectLRU(cacheBytes)
	boundedStorage := filesystem.NewStorage(bounded.NewFilesystem(storageBytes, storageFiles), boundedCache)
	boundedFilesystem := bounded.NewFilesystem(filesystemBytes, filesystemFiles)

	// Clone the repository
	repo, err := git.CloneContext(ctx, boundedStorage, boundedFilesystem, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRetrievalFailed, err)
	}

	// Checkout the working tree to the specified revision
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRetrievalFailed, err)
	}

	// Without this ResolveRevision() would only work for default branch (e.g. master)
	if err := repo.Fetch(&git.FetchOptions{
		RefSpecs: []config.RefSpec{"refs/*:refs/*"},
	}); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRetrievalFailed, err)
	}

	hash, err := repo.ResolveRevision(plumbing.Revision(Revision))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRetrievalFailed, err)
	}

	if err := worktree.Checkout(&git.CheckoutOptions{
		Hash: *hash,
	}); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRetrievalFailed, err)
	}

	return &Git{worktree: worktree}, nil
}

func (g Git) Stat(ctx context.Context, path string) (*fs.FileInfo, error) {
	stat, err := g.worktree.Filesystem.Stat(path)
	if err != nil {
		return nil, err
	}
	return &fs.FileInfo{IsDir: stat.IsDir()}, nil
}

func (g Git) Get(ctx context.Context, path string) ([]byte, error) {
	file, err := g.worktree.Filesystem.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if strings.Contains(err.Error(), "cannot open directory") {
			return nil, fs.ErrNormalizedIsADirectory
		}

		return nil, fmt.Errorf("%w: %v", ErrRetrievalFailed, err)
	}
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRetrievalFailed, err)
	}

	return fileBytes, nil
}

func (g Git) ReadDir(ctx context.Context, path string) ([]string, error) {
	stat, err := g.worktree.Filesystem.Stat(path)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, syscall.ENOTDIR
	}
	infos, err := g.worktree.Filesystem.ReadDir(path)
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, os.ErrNotExist
	}
	var entries []string
	for _, info := range infos {
		entries = append(entries, info.Name())
	}

	return entries, nil
}

func (g Git) Join(elem ...string) string {
	return path.Join(elem...)
}
