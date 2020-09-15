package git

import (
	"context"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"io/ioutil"
	"regexp"
)

type Locator struct {
	URL      string
	Path     string
	Revision string
}

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
		Revision: "master",
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
	storer := memory.NewStorage()
	fs := memfs.New()

	// Clone the repository
	repo, err := git.CloneContext(ctx, storer, fs, &git.CloneOptions{
		URL:   locator.URL,
		Depth: 1,
	})
	if err != nil {
		return nil, err
	}

	// Checkout the working tree to the specified revision
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	hash, err := repo.ResolveRevision(plumbing.Revision(locator.Revision))
	if err != nil {
		return nil, err
	}

	if err := worktree.Checkout(&git.CheckoutOptions{
		Hash: *hash,
	}); err != nil {
		return nil, err
	}

	// Read the file from the working tree
	file, err := worktree.Filesystem.Open(locator.Path)
	if err != nil {
		return nil, err
	}

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return fileBytes, nil
}
