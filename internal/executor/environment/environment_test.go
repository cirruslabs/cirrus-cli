package environment_test

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestProjectSpecific ensures that we collect Git metadata about the project properly.
func TestProjectSpecific(t *testing.T) {
	dir := testutil.TempDir(t)

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	workTree, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy file and add it to the staging area
	const testFile = "test.txt"
	if err := os.WriteFile(filepath.Join(dir, testFile), []byte("test\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = workTree.Add(testFile)
	if err != nil {
		t.Fatal(err)
	}

	// Commit the files in the staging area
	const commitMessage = "Initial commit"
	commitHash, err := workTree.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Charlie Root",
			Email: "root@localhost",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create a branch that points to the commit we've just created
	const branchName = "master"
	refName := fmt.Sprintf("refs/heads/%s", branchName)
	ref := plumbing.NewHashReference(plumbing.ReferenceName(refName), commitHash)
	if err := repo.Storer.SetReference(ref); err != nil {
		t.Fatal(err)
	}

	// Create a tag that points to the commit we've just created
	_, err = repo.CreateTag("v1.2.3", commitHash, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Configure a GitHub remote
	const repoOwner, repoName = "cirruslabs", "example"
	cloneURL := fmt.Sprintf("git@github.com:%s/%s.git", repoOwner, repoName)
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{cloneURL},
	})
	if err != nil {
		t.Fatal(err)
	}

	expectedEnv := map[string]string{
		"CIRRUS_REPO_CLONE_URL": cloneURL,
		"CIRRUS_REPO_OWNER":     repoOwner,
		"CIRRUS_REPO_NAME":      repoName,
		"CIRRUS_REPO_FULL_NAME": fmt.Sprintf("%s/%s", repoOwner, repoName),

		"CIRRUS_BRANCH": branchName,

		"CIRRUS_CHANGE_IN_REPO": commitHash.String(),
		"CIRRUS_CHANGE_MESSAGE": commitMessage,

		"CIRRUS_TAG": "v1.2.3",
	}

	assert.Equal(t, expectedEnv, environment.ProjectSpecific(dir))
}

// TestMerge ensures that the environments are merged in the correct order (i.e. first argument
// has the lowest priority while the last argument has the highest priority).
func TestMerge(t *testing.T) {
	merged := environment.Merge(
		map[string]string{"A": "low", "B": "low"},
		map[string]string{"B": "med", "C": "med"},
		map[string]string{"A": "1", "B": "2", "C": "3"},
	)

	assert.Equal(t, map[string]string{"A": "1", "B": "2", "C": "3"}, merged)
}
