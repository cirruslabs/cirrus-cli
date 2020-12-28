package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

var ErrAgentDownloadFailed = errors.New("failed to download agent")

func RetrieveBinary(
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
