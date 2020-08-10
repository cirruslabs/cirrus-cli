package environment

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"github.com/go-git/go-git/v5"
	"github.com/google/uuid"
	"path"
	"strconv"
	"strings"
)

func Merge(opts ...map[string]string) map[string]string {
	result := make(map[string]string)

	for _, opt := range opts {
		for key, value := range opt {
			result[key] = value
		}
	}

	return result
}

func Static() map[string]string {
	return map[string]string{
		"CIRRUS_WORKING_DIR": path.Join(instance.WorkingVolumeMountpoint, instance.WorkingVolumeWorkingDir),

		"CI":                     "true",
		"CONTINUOUS_INTEGRATION": "true",
		"CIRRUS_CI":              "true",
		"CIRRUS_ENVIRONMENT":     "CLI",

		// The semantics is clearly a bit shaky here, but from the usability standpoint
		// this is better than not setting these at all (or setting them "false") since
		// more tasks will be able to run out-of-the-box.
		"CIRRUS_USER_COLLABORATOR": "true",
		"CIRRUS_USER_PERMISSION":   "write",
	}
}

func BuildID() map[string]string {
	return map[string]string{
		"CIRRUS_BUILD_ID": "CLI-" + uuid.New().String(),
	}
}

func NodeInfo(nodeIndex, nodeTotal int64) map[string]string {
	return map[string]string{
		"CI_NODE_INDEX": strconv.FormatInt(nodeIndex, 10),
		"CI_NODE_TOTAL": strconv.FormatInt(nodeTotal, 10),
	}
}

func TaskInfo(taskName string, taskID int64) map[string]string {
	return map[string]string{
		"CIRRUS_TASK_NAME": taskName,
		"CIRRUS_TASK_ID":   strconv.FormatInt(taskID, 10),
	}
}

func ProjectSpecific(projectDir string) map[string]string {
	result := make(map[string]string)

	repo, err := git.PlainOpen(projectDir)
	if err != nil {
		return result
	}

	addRemoteVariables(result, repo)
	addBranchVariables(result, repo)
	addCommitVariables(result, repo)

	return result
}

func addRemoteVariables(envMap map[string]string, repo *git.Repository) {
	remote, err := repo.Remote("origin")
	if err != nil {
		return
	}

	if len(remote.Config().URLs) < 1 {
		return
	}

	url := remote.Config().URLs[0]

	// Cut URL's prefix if it looks like a GitHub's one or bail out
	cutURL := url
	const httpsPrefix = "https://github.com/"
	const sshPrefix = "git@github.com:"

	switch {
	case strings.HasPrefix(cutURL, httpsPrefix):
		cutURL = strings.TrimPrefix(cutURL, httpsPrefix)
	case strings.HasPrefix(cutURL, sshPrefix):
		cutURL = strings.TrimPrefix(cutURL, sshPrefix)
	default:
		return
	}

	// Cut URL's suffix
	const gitSuffix = ".git"
	cutURL = strings.TrimSuffix(cutURL, gitSuffix)

	// Extract the repository owner and name
	const cleanURLParts = 2

	parts := strings.Split(cutURL, "/")
	if len(parts) != cleanURLParts {
		return
	}

	envMap["CIRRUS_REPO_CLONE_URL"] = url
	repoOwner, repoName := parts[0], parts[1]
	envMap["CIRRUS_REPO_OWNER"] = repoOwner
	envMap["CIRRUS_REPO_NAME"] = repoName
	envMap["CIRRUS_REPO_FULL_NAME"] = fmt.Sprintf("%s/%s", repoOwner, repoName)
}

func addBranchVariables(envMap map[string]string, repo *git.Repository) {
	head, err := repo.Head()
	if err != nil {
		return
	}

	if !head.Name().IsBranch() {
		return
	}

	envMap["CIRRUS_BRANCH"] = head.Name().Short()
}

func addCommitVariables(envMap map[string]string, repo *git.Repository) {
	head, err := repo.Head()
	if err != nil {
		return
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return
	}

	envMap["CIRRUS_CHANGE_IN_REPO"] = commit.Hash.String()
	envMap["CIRRUS_CHANGE_MESSAGE"] = commit.Message
}
