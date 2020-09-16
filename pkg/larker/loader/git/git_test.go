package git_test

import (
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
			Revision: "master",
		}},
		{"parses path", "github.com/some-org/some-repo/dir/some.star", &git.Locator{
			URL:      "https://github.com/some-org/some-repo.git",
			Path:     "dir/some.star",
			Revision: "master",
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
			Revision: "master",
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
