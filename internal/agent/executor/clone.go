package executor

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/environment"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	gitclient "github.com/go-git/go-git/v5/plumbing/transport/client"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func CloneRepository(
	ctx context.Context,
	logUploader io.Writer,
	env *environment.Environment,
) bool {
	logUploader.Write([]byte("Using built-in Git...\n"))

	working_dir := env.Get("CIRRUS_WORKING_DIR")
	change := env.Get("CIRRUS_CHANGE_IN_REPO")
	branch := env.Get("CIRRUS_BRANCH")
	pr_number, is_pr := env.Lookup("CIRRUS_PR")
	tag, is_tag := env.Lookup("CIRRUS_TAG")
	is_clone_modules := env.Get("CIRRUS_CLONE_SUBMODULES") == "true"

	useMergeRef := env.Get("CIRRUS_RESOLUTION_STRATEGY") == "MERGE_FOR_PRS"

	clone_url := env.Get("CIRRUS_REPO_CLONE_URL")
	if _, has_clone_token := env.Lookup("CIRRUS_REPO_CLONE_TOKEN"); has_clone_token {
		clone_url = env.ExpandText("https://x-access-token:${CIRRUS_REPO_CLONE_TOKEN}@${CIRRUS_REPO_CLONE_HOST}/${CIRRUS_REPO_FULL_NAME}.git")
	}

	clone_depth := 0
	if depth_str, ok := env.Lookup("CIRRUS_CLONE_DEPTH"); ok {
		clone_depth, _ = strconv.Atoi(depth_str)
	}
	if clone_depth > 0 {
		logUploader.Write([]byte(fmt.Sprintf("\nLimiting clone depth to %d!", clone_depth)))
	}

	customClient := &http.Client{
		Timeout: 900 * time.Second,
	}
	gitclient.InstallProtocol("https", githttp.NewClient(customClient))
	gitclient.InstallProtocol("http", githttp.NewClient(customClient))

	var repo *git.Repository
	var err error

	tagsOption := git.NoTags
	if env.Get("CIRRUS_CLONE_TAGS") == "true" {
		tagsOption = git.AllTags
	}
	if is_pr {
		repo, err = git.PlainInit(working_dir, false)
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to init repository: %s!", err)))
			return false
		}
		remoteConfig := &config.RemoteConfig{
			Name: "origin",
			URLs: []string{clone_url},
		}
		if _, err := repo.CreateRemote(remoteConfig); err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to create remote: %s!", err)))
			return false
		}

		var refSpec string

		if useMergeRef {
			refSpec = fmt.Sprintf("+refs/pull/%s/merge:refs/remotes/origin/pull/%[1]s", pr_number)
			if clone_depth > 0 {
				// increase by one since we are cloning with an extra "merge" commit from GH
				clone_depth = clone_depth + 1
			}
		} else {
			refSpec = fmt.Sprintf("+refs/pull/%s/head:refs/remotes/origin/pull/%[1]s", pr_number)
		}

		logUploader.Write([]byte(fmt.Sprintf("\nFetching %s...\n", refSpec)))
		fetchOptions := &git.FetchOptions{
			RemoteName: remoteConfig.Name,
			RefSpecs:   []config.RefSpec{config.RefSpec(refSpec)},
			Tags:       tagsOption,
			Progress:   logUploader,
			Depth:      clone_depth,
		}
		err = repo.FetchContext(ctx, fetchOptions)
		if err != nil && retryableCloneError(err) {
			logUploader.Write([]byte(fmt.Sprintf("\nFetch failed: %s!", err)))
			logUploader.Write([]byte("\nRe-trying to fetch..."))
			err = repo.FetchContext(ctx, fetchOptions)
		}
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed fetch: %s!", err)))
			return false
		}

		workTree, err := repo.Worktree()
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to get work tree: %s!", err)))
			return false
		}

		if useMergeRef {
			checkoutOptions := git.CheckoutOptions{
				Branch: plumbing.ReferenceName(fmt.Sprintf("refs/remotes/origin/pull/%[1]s", pr_number)),
			}
			logUploader.Write([]byte(fmt.Sprintf("\nChecking out %s...", checkoutOptions.Branch)))
			err = workTree.Checkout(&checkoutOptions)
			if err != nil {
				logUploader.Write([]byte(fmt.Sprintf("\nFailed to checkout %s: %s!", checkoutOptions.Branch, err)))
				return false
			}
		} else {
			checkoutOptions := git.CheckoutOptions{
				Hash: plumbing.NewHash(change),
			}
			logUploader.Write([]byte(fmt.Sprintf("\nChecking out %s...", checkoutOptions.Hash)))
			err = workTree.Checkout(&checkoutOptions)
			if err != nil {
				logUploader.Write([]byte(fmt.Sprintf("\nFailed to checkout %s: %s!", checkoutOptions.Hash, err)))
				return false
			}
		}
	} else {
		cloneOptions := git.CloneOptions{
			URL:      clone_url,
			Progress: logUploader,
			Depth:    clone_depth,
		}
		if !is_tag {
			cloneOptions.Tags = tagsOption
		}

		if is_tag {
			cloneOptions.SingleBranch = true
			cloneOptions.ReferenceName = plumbing.ReferenceName(fmt.Sprintf("refs/tags/%s", tag))
		} else {
			cloneOptions.SingleBranch = true
			cloneOptions.ReferenceName = plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch))
		}
		logUploader.Write([]byte(fmt.Sprintf("\nCloning %s...\n", cloneOptions.ReferenceName)))

		repo, err = git.PlainCloneContext(ctx, working_dir, false, &cloneOptions)

		if err != nil && retryableCloneError(err) {
			logUploader.Write([]byte(fmt.Sprintf("\nRetryable error '%s' while cloning! Trying again...", err)))
			os.RemoveAll(working_dir)
			EnsureFolderExists(working_dir)
			repo, err = git.PlainClone(working_dir, false, &cloneOptions)
		}

		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "timeout") || strings.Contains(strings.ToLower(err.Error()), "timed out") {
				logUploader.Write([]byte("\nFailed to clone because of a timeout from Git server!"))
			} else {
				logUploader.Write([]byte(fmt.Sprintf("\nFailed to clone: %s!", err)))
			}
			return false
		}
	}

	ref, err := repo.Head()
	if err != nil {
		logUploader.Write([]byte("\nFailed to get HEAD information!"))
		return false
	}

	if !useMergeRef && ref.Hash() != plumbing.NewHash(change) {
		logUploader.Write([]byte(fmt.Sprintf("\nHEAD is at %s.", ref.Hash())))
		logUploader.Write([]byte(fmt.Sprintf("\nHard resetting to %s...", change)))

		workTree, err := repo.Worktree()
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to get work tree: %s!", err)))
			return false
		}

		err = workTree.Reset(&git.ResetOptions{
			Commit: plumbing.NewHash(change),
			Mode:   git.HardReset,
		})
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to force reset to %s: %s!", change, err)))
			return false
		}
	} else if useMergeRef {
		logUploader.Write([]byte(fmt.Sprintf("\n\"Merge for PRs\" config resolution strategy enabled, " +
			"skipping hard reset to preserve the merge commit")))
	}

	if is_clone_modules {
		logUploader.Write([]byte("\nUpdating submodules..."))

		workTree, err := repo.Worktree()
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to get work tree: %s!", err)))
			return false
		}

		submodules, err := workTree.Submodules()
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to get submodules: %s!", err)))
			return false
		}

		opts := &git.SubmoduleUpdateOptions{
			Init:              true,
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		}

		for _, sub := range submodules {
			if err := sub.UpdateContext(ctx, opts); err != nil {
				logUploader.Write([]byte(fmt.Sprintf("\nFailed to update submodule %q: %s!",
					sub.Config().Name, err)))
				return false
			}
		}

		logUploader.Write([]byte("\nSucessfully updated submodules!"))
	}

	ref, err = repo.Head()
	if err != nil {
		logUploader.Write([]byte("\nFailed to get HEAD information!"))
		return false
	}

	logUploader.Write([]byte(fmt.Sprintf("\nChecked out %s on %s branch.",
		ref.Hash().String(), branch)))
	logUploader.Write([]byte("\nSuccessfully cloned!"))

	return true
}
