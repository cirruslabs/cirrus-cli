package github

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/google/go-github/v53/github"
	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/oauth2"
	"net/http"
	"os"
	"path"
	"syscall"
)

var ErrAPI = errors.New("failed to communicate with the GitHub API")

type GitHub struct {
	token     string
	owner     string
	repo      string
	reference string

	contentsCache  *lru.Cache
	fileInfosCache *lru.Cache

	apiCallCount uint64
}

type Contents struct {
	File      *github.RepositoryContent
	Directory []*github.RepositoryContent
}

func New(owner, repo, reference, token string) (*GitHub, error) {
	contentsCache, err := lru.New(16)
	if err != nil {
		return nil, err
	}
	fileInfosCache, err := lru.New(1024)
	if err != nil {
		return nil, err
	}

	return &GitHub{
		token:     token,
		owner:     owner,
		repo:      repo,
		reference: reference,

		contentsCache:  contentsCache,
		fileInfosCache: fileInfosCache,
	}, nil
}

func (gh *GitHub) APICallCount() uint64 {
	return gh.apiCallCount
}

func (gh *GitHub) Stat(ctx context.Context, path string) (*fs.FileInfo, error) {
	cachedFileInfo, ok := gh.fileInfosCache.Get(path)
	if ok {
		return cachedFileInfo.(*fs.FileInfo), nil
	}

	_, directoryContent, err := gh.getContentsWrapper(ctx, path)
	if err != nil {
		return nil, err
	}

	if directoryContent != nil {
		return &fs.FileInfo{IsDir: true}, nil
	}

	return &fs.FileInfo{IsDir: false}, nil
}

func (gh *GitHub) Get(ctx context.Context, path string) ([]byte, error) {
	fileContent, _, err := gh.getContentsWrapper(ctx, path)
	if err != nil {
		return nil, err
	}

	// Simulate os.Read() behavior in case the supplied path points to a directory
	if fileContent == nil {
		return nil, fs.ErrNormalizedIsADirectory
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

func (gh *GitHub) Join(elem ...string) string {
	return path.Join(elem...)
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
	contents, ok := gh.contentsCache.Get(path)
	if ok {
		return contents.(*Contents).File, contents.(*Contents).Directory, nil
	}

	gh.apiCallCount++

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

	if fileContent != nil {
		gh.fileInfosCache.ContainsOrAdd(path, &fs.FileInfo{IsDir: false})
	}
	for _, directoryEntry := range directoryContent {
		if directoryEntry.Type == nil || directoryEntry.Path == nil {
			continue
		}

		var fileInfo fs.FileInfo

		switch *directoryEntry.Type {
		case "file":
			fileInfo.IsDir = false
		case "dir":
			fileInfo.IsDir = true
		default:
			continue
		}

		gh.fileInfosCache.ContainsOrAdd(*directoryEntry.Path, &fileInfo)
	}

	gh.contentsCache.Add(path, &Contents{File: fileContent, Directory: directoryContent})

	return fileContent, directoryContent, nil
}
