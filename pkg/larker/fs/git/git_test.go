package git_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/git"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/githubfixture"
	"testing"
)

func TestGitHubFixture(t *testing.T) {
	gitFS, err := git.New(context.Background(), githubfixture.URL, githubfixture.Reference)
	if err != nil {
		t.Fatal(err)
	}

	githubfixture.Run(t, gitFS)
}
