package loader

import (
	"context"
	"errors"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/git"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/github"
	"regexp"
)

type localLocation struct {
	Path string
}

type gitLocation struct {
	URL      string
	Revision string
	Path     string
}

type gitHubLocation struct {
	Owner    string
	Name     string
	Revision string
	Path     string
}

var (
	ErrRetrievalFailed     = errors.New("failed to retrieve a file from Git repository")
	ErrFileNotFound        = errors.New("file not found in a Git repository")
	ErrUnsupportedLocation = errors.New("unsupported location")
)

const (
	// Captures the path after / in non-greedy manner.
	optionalPath = `(?:/(?P<path>.*?))?`

	// Captures the revision after @ in non-greedy manner.
	optionalRevision = `(?:@(?P<revision>.*))?`
)

var (
	githubRegexVariant = regexp.MustCompile(
		`^(?P<root>github\.com/(?P<owner>.*?)/(?P<name>.*?))` + optionalPath + optionalRevision + `$`,
	)

	genericGitRegexVariant = regexp.MustCompile(`^(?P<root>.*?)\.git` + optionalPath + optionalRevision + `$`)
)

func parseLocation(module string) interface{} {
	matches := githubRegexVariant.FindStringSubmatch(module)
	if matches != nil {
		owner := matches[githubRegexVariant.SubexpIndex("owner")]
		name := matches[githubRegexVariant.SubexpIndex("name")]

		revision := matches[githubRegexVariant.SubexpIndex("revision")]
		if revision == "" {
			revision = "main"
		}

		modulePath := matches[githubRegexVariant.SubexpIndex("path")]
		if modulePath == "" {
			modulePath = "lib.star"
		}

		return gitHubLocation{
			Owner:    owner,
			Name:     name,
			Revision: revision,
			Path:     modulePath,
		}
	}

	matches = genericGitRegexVariant.FindStringSubmatch(module)
	if matches != nil {
		revision := matches[genericGitRegexVariant.SubexpIndex("revision")]
		if revision == "" {
			revision = "main"
		}

		modulePath := matches[genericGitRegexVariant.SubexpIndex("path")]
		if modulePath == "" {
			modulePath = "lib.star"
		}

		return gitLocation{
			URL:      "https://" + matches[genericGitRegexVariant.SubexpIndex("root")] + ".git",
			Revision: revision,
			Path:     modulePath,
		}
	}

	return localLocation{Path: module}
}

func findModuleFS(
	ctx context.Context,
	currentFS fs.FileSystem,
	env map[string]string,
	module string,
) (fs.FileSystem, string, error) {
	return findLocatorFS(ctx, currentFS, env, parseLocation(module))
}
func findLocatorFS(
	ctx context.Context,
	currentFS fs.FileSystem,
	env map[string]string,
	location interface{},
) (fs.FileSystem, string, error) {
	switch l := location.(type) {
	case gitHubLocation:
		token := env["CIRRUS_REPO_CLONE_TOKEN"]

		ghFS, err := github.New(l.Owner, l.Name, l.Revision, token)
		if err != nil {
			return nil, "", err
		}
		return ghFS, l.Path, nil
	case gitLocation:
		gitFS, err := git.New(ctx, l.URL, l.Revision)
		if err != nil {
			return nil, "", err
		}
		return gitFS, l.Path, nil
	case localLocation:
		return currentFS, l.Path, nil
	default:
		return nil, "", ErrUnsupportedLocation
	}
}
