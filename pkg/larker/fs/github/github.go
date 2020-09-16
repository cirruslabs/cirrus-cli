package github

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/google/go-github/v32/github"
	"syscall"
)

var ErrAPI = errors.New("failed to communicate with the GitHub API")

type GitHub struct {
	client    *github.Client
	owner     string
	repo      string
	reference string
}

func New(owner, repo, reference, token string) *GitHub {
	return &GitHub{
		client:    github.NewClient(nil),
		owner:     owner,
		repo:      repo,
		reference: reference,
	}
}

func (gh *GitHub) Get(ctx context.Context, path string) ([]byte, error) {
	fileContent, _, _, err := gh.client.Repositories.GetContents(ctx, gh.owner, gh.repo, path,
		&github.RepositoryContentGetOptions{
			Ref: gh.reference,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAPI, err)
	}

	// Simulate os.Read() behavior in case the supplied path points to a directory
	if fileContent == nil {
		return nil, syscall.EISDIR
	}

	fileBytes, err := base64.StdEncoding.DecodeString(*fileContent.Content)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAPI, err)
	}

	return fileBytes, nil
}
