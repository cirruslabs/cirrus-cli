//nolint:testpackage // testing private methods
package resolver

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/dummy"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		Name               string
		Module             string
		ExpectedGitLocator interface{}
	}{
		// GitHub
		{"defaults to lib.star", "github.com/some-org/some-repo", gitHubLocation{
			Owner:    "some-org",
			Name:     "some-repo",
			Path:     "lib.star",
			Revision: "main",
		}},
		{"parses path", "github.com/some-org/some-repo/dir/some.star", gitHubLocation{
			Owner:    "some-org",
			Name:     "some-repo",
			Path:     "dir/some.star",
			Revision: "main",
		}},
		{"parses revision", "github.com/some-org/some-repo@da39a3ee5e6b4b0d3255bfef95601890afd80709", gitHubLocation{
			Owner:    "some-org",
			Name:     "some-repo",
			Path:     "lib.star",
			Revision: "da39a3ee5e6b4b0d3255bfef95601890afd80709",
		}},
		{"parses path and revision", "github.com/some-org/some-repo/dir/some.star@da39a3ee", gitHubLocation{
			Owner:    "some-org",
			Name:     "some-repo",
			Path:     "dir/some.star",
			Revision: "da39a3ee",
		}},
		// Other Git hosting services (with the ".git" hint)
		{"parses .git hint", "gitlab.com/some-org/some-repo.git/some.star", gitLocation{
			URL:      "https://gitlab.com/some-org/some-repo.git",
			Path:     "some.star",
			Revision: "main",
		}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			assert.Equal(t, testCase.ExpectedGitLocator, parseLocation(testCase.Module))
		})
	}
}

func TestRetrieve(t *testing.T) {
	testCases := []struct {
		Name    string
		Locator interface{}
	}{
		{
			"default branch",
			gitHubLocation{
				Owner:    "cirrus-modules",
				Name:     "helpers",
				Path:     "lib.star",
				Revision: "main",
			},
		},
		{"non-default branch",
			gitLocation{
				URL:      "https://github.com/cirrus-modules/helpers",
				Path:     "lib.star",
				Revision: "branch-for-cli-testing",
			},
		},
		{"tag",
			gitHubLocation{
				Owner:    "cirrus-modules",
				Name:     "helpers",
				Path:     "lib.star",
				Revision: "v0.1.0",
			},
		},
		{"hash",
			gitLocation{
				URL:      "https://github.com/cirrus-modules/helpers",
				Path:     "lib.star",
				Revision: "89fb5418c2e5c430baeab7b1bdc5f7be189cc848",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			filesystem, path, err := findLocatorFS(context.Background(), dummy.New(), make(map[string]string), testCase.Locator)
			if err != nil {
				t.Fatal(err)
			}
			result, err := filesystem.Get(context.Background(), path)
			if err != nil {
				t.Fatal(err)
			}
			assert.Contains(t, string(result), "def task(")
		})
	}
}
