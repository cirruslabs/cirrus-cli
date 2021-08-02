package helpers

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
	"github.com/go-git/go-git/v5/utils/merkletrie/filesystem"
	mindex "github.com/go-git/go-git/v5/utils/merkletrie/index"
	"github.com/go-git/go-git/v5/utils/merkletrie/noder"
	"os"
	"strings"
)

// GitDiff is a simplified "git diff" and "git diff --cached" implementation using go-git.
//
// GitDiff closely resembles the implementation of go-git Worktree's Status() method.
func GitDiff(dir string, revision string, cached bool) ([]string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return nil, err
	}

	hash, err := repo.ResolveRevision(plumbing.Revision(revision))
	if err != nil {
		return nil, err
	}

	commit, err := repo.CommitObject(*hash)
	if err != nil {
		return nil, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	from := object.NewTreeRootNode(tree)

	var to noder.Noder

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	if cached {
		idx, err := repo.Storer.Index()
		if err != nil {
			return nil, err
		}

		to = mindex.NewRootNode(idx)
	} else {
		submodules, err := getSubmodulesStatus(worktree)
		if err != nil {
			return nil, err
		}

		to = filesystem.NewRootNode(worktree.Filesystem, submodules)
	}

	// .gitignore support
	changes, err := merkletrie.DiffTree(from, to, diffTreeIsEquals)
	if err != nil {
		return nil, err
	}
	patterns, err := gitignore.ReadPatterns(worktree.Filesystem, nil)
	if err != nil {
		return nil, err
	}
	matcher := gitignore.NewMatcher(patterns)

	var result []string

	for _, change := range changes {
		action, err := change.Action()
		if err != nil {
			return nil, err
		}

		var path string
		var isDir bool

		switch action {
		case merkletrie.Insert:
			fallthrough
		case merkletrie.Modify:
			path = change.To.String()
			isDir = change.To.IsDir()
		case merkletrie.Delete:
			path = change.From.String()
			isDir = change.From.IsDir()
		default:
			continue
		}

		if matcher.Match(strings.Split(path, string(os.PathSeparator)), isDir) {
			continue
		}

		result = append(result, path)
	}

	return result, nil
}
