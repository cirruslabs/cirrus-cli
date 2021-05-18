package git_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/loader/git"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		Name               string
		Module             string
		ExpectedGitLocator *git.Locator
	}{
		// GitHub
		{"defaults to lib.star", "github.com/some-org/some-repo", &git.Locator{
			URL:      "https://github.com/some-org/some-repo.git",
			Path:     "lib.star",
			Revision: "main",
		}},
		{"parses path", "github.com/some-org/some-repo/dir/some.star", &git.Locator{
			URL:      "https://github.com/some-org/some-repo.git",
			Path:     "dir/some.star",
			Revision: "main",
		}},
		{"parses revision", "github.com/some-org/some-repo@da39a3ee5e6b4b0d3255bfef95601890afd80709", &git.Locator{
			URL:      "https://github.com/some-org/some-repo.git",
			Path:     "lib.star",
			Revision: "da39a3ee5e6b4b0d3255bfef95601890afd80709",
		}},
		{"parses path and revision", "github.com/some-org/some-repo/dir/some.star@da39a3ee", &git.Locator{
			URL:      "https://github.com/some-org/some-repo.git",
			Path:     "dir/some.star",
			Revision: "da39a3ee",
		}},
		// Other Git hosting services (with the ".git" hint)
		{"parses .git hint", "gitlab.com/some-org/some-repo.git/some.star", &git.Locator{
			URL:      "https://gitlab.com/some-org/some-repo.git",
			Path:     "some.star",
			Revision: "main",
		}},
		// Other Git hosting services (without the ".git" hint)
		{"fails without .git hint", "gitlab.com/some-org/some-repo", nil},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			assert.Equal(t, testCase.ExpectedGitLocator, git.Parse(testCase.Module))
		})
	}
}

func TestRetrieve(t *testing.T) {
	testCases := []struct {
		Name    string
		Locator *git.Locator
	}{
		{
			"default branch",
			&git.Locator{
				URL:      "https://github.com/cirrus-modules/helpers",
				Path:     "lib.star",
				Revision: "main",
			},
		},
		{"non-default branch",
			&git.Locator{URL: "https://github.com/cirrus-modules/helpers",
				Path:     "lib.star",
				Revision: "branch-for-cli-testing",
			},
		},
		{"tag",
			&git.Locator{URL: "https://github.com/cirrus-modules/helpers",
				Path:     "lib.star",
				Revision: "v0.1.0",
			},
		},
		{"hash",
			&git.Locator{URL: "https://github.com/cirrus-modules/helpers",
				Path:     "lib.star",
				Revision: "89fb5418c2e5c430baeab7b1bdc5f7be189cc848",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			result, err := git.Retrieve(context.Background(), testCase.Locator)
			if err != nil {
				t.Fatal(err)
			}
			assert.Contains(t, string(result), "def task(")
		})
	}
}
