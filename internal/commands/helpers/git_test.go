package helpers_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/commands/helpers"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"
)

func TestGitDiff(t *testing.T) {
	testutil.TempChdir(t)

	// Create a repository
	repo, err := git.PlainInit(".", false)
	if err != nil {
		t.Fatal(err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	// Create a file and commit it
	if err := ioutil.WriteFile("canary", []byte("original content"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := worktree.Add("canary"); err != nil {
		t.Fatal(err)
	}
	originalCommit, err := worktree.Commit("Add canary with original content", &git.CommitOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Modify the file and ensure that GitDiff detects the changes against the HEAD
	if err := ioutil.WriteFile("canary", []byte("modified content"), 0600); err != nil {
		t.Fatal(err)
	}
	affectedFiles, err := helpers.GitDiff(".", "HEAD", false)
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, []string{"canary"}, affectedFiles)

	// Commit the changes
	if _, err := worktree.Add("canary"); err != nil {
		t.Fatal(err)
	}
	if _, err := worktree.Commit("Add canary with modified content", &git.CommitOptions{}); err != nil {
		t.Fatal(err)
	}

	// Revert the file contents to be similar with the first commit and ensure that
	// GitDiff reports no changes when ran against the specific commit
	if err := ioutil.WriteFile("canary", []byte("original content"), 0600); err != nil {
		t.Fatal(err)
	}
	affectedFiles, err = helpers.GitDiff(".", originalCommit.String(), false)
	if err != nil {
		t.Fatal(err)
	}
	require.Empty(t, affectedFiles)
}
