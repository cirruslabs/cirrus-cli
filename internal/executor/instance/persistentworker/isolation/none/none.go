package none

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/agent"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/otiai10/copy"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strconv"
)

var (
	ErrPopulateFailed = errors.New("failed to populate working directory")
)

type PersistentWorkerInstance struct {
	tempDir string
	cleanup func() error
}

func New() (*PersistentWorkerInstance, error) {
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

func (pwi *PersistentWorkerInstance) Run(ctx context.Context, config *runconfig.RunConfig) (err error) {
	// Retrieve the agent's binary
	agentPath, err := agent.RetrieveBinary(ctx, config.GetAgentVersion(), runtime.GOOS, runtime.GOARCH)
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
		"-pre-created-working-dir",
		pwi.tempDir,
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
	return cmd.Run()
}

func (pwi *PersistentWorkerInstance) WorkingDirectory(projectDir string, dirtyMode bool) string {
	if dirtyMode {
		return projectDir
	}

	return pwi.tempDir
}

func (pwi *PersistentWorkerInstance) Close() error {
	return pwi.cleanup()
}
