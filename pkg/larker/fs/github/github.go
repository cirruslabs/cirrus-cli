package github

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
	"net/http"
	"os"
	"syscall"
)

var ErrAPI = errors.New("failed to communicate with the GitHub API")

type GitHub struct {
	token     string
	owner     string
	repo      string
	reference string
}

type IsDir bool

func (isDir IsDir) IsDir() bool {
	return bool(isDir)
}

func New(owner, repo, reference, token string) *GitHub {
	return &GitHub{
		token:     token,
		owner:     owner,
		repo:      repo,
		reference: reference,
	}
}

func (gh *GitHub) Stat(ctx context.Context, path string) (fs.FileInfo, error) {
	_, directoryContent, err := gh.getContentsWrapper(ctx, path)
	if err != nil {
		return nil, err
	}

	if directoryContent != nil {
		return IsDir(true), nil
	}

	return IsDir(false), nil
}

func (gh *GitHub) Get(ctx context.Context, path string) ([]byte, error) {
	fileContent, _, err := gh.getContentsWrapper(ctx, path)
	if err != nil {
		return nil, err
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

func (gh *GitHub) ReadDir(ctx context.Context, path string) ([]string, error) {
	_, directoryContent, err := gh.getContentsWrapper(ctx, path)
	if err != nil {
		return nil, err
	}

	// Simulate ioutil.ReadDir() behavior in case the supplied path points to a file
	if directoryContent == nil {
		return nil, syscall.ENOTDIR
	}

	var entries []string
	for _, fileContent := range directoryContent {
		entries = append(entries, *fileContent.Name)
	}

	return entries, nil
}

func (gh *GitHub) client(ctx context.Context) *github.Client {
	var client *http.Client

	if gh.token != "" {
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: gh.token,
		})
		client = oauth2.NewClient(ctx, tokenSource)
	}

	return github.NewClient(client)
}

func (gh *GitHub) getContentsWrapper(
	ctx context.Context,
	path string,
) (*github.RepositoryContent, []*github.RepositoryContent, error) {
	fileContent, directoryContent, resp, err := gh.client(ctx).Repositories.GetContents(ctx, gh.owner, gh.repo, path,
		&github.RepositoryContentGetOptions{
			Ref: gh.reference,
		},
	)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil, os.ErrNotExist
		}

		return nil, nil, fmt.Errorf("%w: %v", ErrAPI, err)
	}

	return fileContent, directoryContent, nil
}
