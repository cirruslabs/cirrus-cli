package instance

import (
	"context"
	"errors"
	"fmt"
	"github.com/otiai10/copy"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

var (
	ErrPopulateFailed      = errors.New("failed to populate working directory")
	ErrAgentDownloadFailed = errors.New("failed to download agent")
)

type PersistentWorkerInstance struct {
	tempDir string
	cleanup func() error
}

func NewPersistentWorkerInstance() (*PersistentWorkerInstance, error) {
	// Create a working directory that will be used if no dirty mode is requested in Run()
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}

	return &PersistentWorkerInstance{
		tempDir: tempDir,
		cleanup: func() error {
			return os.RemoveAll(tempDir)
		},
	}, nil
}

func (pwi *PersistentWorkerInstance) Run(ctx context.Context, config *RunConfig) (err error) {
	// Retrieve the agent's binary
	agentPath, err := RetrieveAgentBinary(ctx, config.GetAgentVersion(), runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, agentPath,
		"-api-endpoint",
		config.DirectEndpoint,
		"-server-token",
		config.ServerSecret,
		"-client-token",
		config.ClientSecret,
		"-task-id",
		strconv.FormatInt(config.TaskID, 10),
	)

	// Determine the working directory for the agent
	if config.DirtyMode {
		cmd.Dir = config.ProjectDir
	} else {
		// Populate the working directory
		if config.ProjectDir != "" {
			if err := copy.Copy(config.ProjectDir, pwi.tempDir); err != nil {
				return fmt.Errorf("%w: while copying %s's contents into %s: %v",
					ErrPopulateFailed, config.ProjectDir, pwi.tempDir, err)
			}
		}

		cmd.Dir = pwi.tempDir
	}

	// Run the agent
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (pwi *PersistentWorkerInstance) WorkingDirectory(projectDir string, dirtyMode bool) string {
	if dirtyMode {
		return projectDir
	}

	return pwi.tempDir
}

func RetrieveAgentBinary(
	ctx context.Context,
	agentVersion string,
	agentOS string,
	agentArchitecture string,
) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	agentCacheDir := filepath.Join(cacheDir, "cirrus", "agent")

	if err := os.MkdirAll(agentCacheDir, 0700); err != nil {
		return "", err
	}

	var agentSuffix string
	if agentOS == "windows" {
		agentSuffix = ".exe"
	}

	agentPath := filepath.Join(
		agentCacheDir,
		fmt.Sprintf("cirrus-ci-agent-%s-%s-%s%s", agentVersion, agentOS, agentArchitecture, agentSuffix),
	)

	// Agent found in the cache
	_, err = os.Stat(agentPath)
	if err == nil {
		return agentPath, nil
	}

	tmpAgentFile, err := ioutil.TempFile(agentCacheDir, "")
	if err != nil {
		return "", err
	}

	// Download the agent
	agentURL := fmt.Sprintf("https://github.com/cirruslabs/cirrus-ci-agent/releases/download/v%s/agent-%s-%s%s",
		agentVersion, runtime.GOOS, runtime.GOARCH, agentSuffix)

	req, err := http.NewRequestWithContext(ctx, "GET", agentURL, http.NoBody)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: got HTTP code %d when downloading %s",
			ErrAgentDownloadFailed, resp.StatusCode, agentURL)
	}

	_, err = io.Copy(tmpAgentFile, resp.Body)
	if err != nil {
		return "", err
	}

	// Make the agent binary executable
	if err := os.Chmod(tmpAgentFile.Name(), 0500); err != nil {
		return "", err
	}

	if err := tmpAgentFile.Close(); err != nil {
		return "", err
	}

	// Move the agent to it's final destination
	if err := os.Rename(tmpAgentFile.Name(), agentPath); err != nil {
		// Already moved by another persistent worker instance?
		if _, err := os.Stat(agentPath); err != nil {
			return agentPath, nil
		}

		return "", err
	}

	return agentPath, nil
}
