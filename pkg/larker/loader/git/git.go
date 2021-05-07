package git

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/loader/git/bounded"
	"github.com/docker/go-units"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"io/ioutil"
	"os"
	"regexp"
)

type Locator struct {
	URL      string
	Path     string
	Revision string
}

var (
	ErrRetrievalFailed = errors.New("failed to retrieve a file from Git repository")
	ErrFileNotFound    = errors.New("file not found in a Git repository")
)

const (
	// Captures the path after / in non-greedy manner.
	optionalPath = `(?:/(?P<path>.*?))?`

	// Captures the revision after @ in non-greedy manner.
	optionalRevision = `(?:@(?P<revision>.*))?`
)

var regexVariants = []*regexp.Regexp{
	// GitHub
	regexp.MustCompile(`^(?P<root>github\.com/.*?/.*?)` + optionalPath + optionalRevision + `$`),
	// Other Git hosting services
	regexp.MustCompile(`^(?P<root>.*?)\.git` + optionalPath + optionalRevision + `$`),
}

func Parse(module string) *Locator {
	result := &Locator{
		Path:     "lib.star",
		Revision: "main",
	}

	for _, regex := range regexVariants {
		matches := regex.FindStringSubmatch(module)
		if matches == nil {
			continue
		}

		result.URL = "https://" + matches[regex.SubexpIndex("root")] + ".git"

		path := matches[regex.SubexpIndex("path")]
		if path != "" {
			result.Path = path
		}

		revision := matches[regex.SubexpIndex("revision")]
		if revision != "" {
			result.Revision = revision
		}

		return result
	}

	return nil
}

func Retrieve(ctx context.Context, locator *Locator) ([]byte, error) {
	const (
		cacheBytes = 1 * units.MiB

		storageBytes = 4 * units.MiB
		storageFiles = 512

		filesystemBytes = 4 * units.MiB
		filesystemFiles = 512
	)

	boundedCache := cache.NewObjectLRU(cacheBytes)
	boundedStorage := filesystem.NewStorage(bounded.NewFilesystem(storageBytes, storageFiles), boundedCache)
	boundedFilesystem := bounded.NewFilesystem(filesystemBytes, filesystemFiles)

	// Clone the repository
	repo, err := git.CloneContext(ctx, boundedStorage, boundedFilesystem, &git.CloneOptions{
		URL: locator.URL,
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

	hash, err := repo.ResolveRevision(plumbing.Revision(locator.Revision))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRetrievalFailed, err)
	}

	if err := worktree.Checkout(&git.CheckoutOptions{
		Hash: *hash,
	}); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRetrievalFailed, err)
	}

	// Read the file from the working tree
	file, err := worktree.Filesystem.Open(locator.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %v", ErrFileNotFound, err)
		}

		return nil, fmt.Errorf("%w: %v", ErrRetrievalFailed, err)
	}

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRetrievalFailed, err)
	}

	return fileBytes, nil
}
