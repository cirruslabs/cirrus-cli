package none

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/agent"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/pwdir"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/logger"
	"github.com/otiai10/copy"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

var (
	ErrPopulateFailed = errors.New("failed to populate working directory")
)

type PersistentWorkerInstance struct {
	logger  logger.Lightweight
	tempDir string
	cleanup func() error
}

func New(opts ...Option) (*PersistentWorkerInstance, error) {
	// Create a working directory that will be used if no dirty mode is requested in Run()
	tempDir, err := pwdir.StaticTempDirWithDynamicFallback()
	if err != nil {
		return nil, err
	}

	pwi := &PersistentWorkerInstance{
		tempDir: tempDir,
		cleanup: func() error {
			return os.RemoveAll(tempDir)
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(pwi)
	}

	// Apply default options (to cover those that weren't specified)
	if pwi.logger == nil {
		pwi.logger = &logger.LightweightStub{}
	}

	return pwi, nil
}

func (pwi *PersistentWorkerInstance) Run(ctx context.Context, config *runconfig.RunConfig) (err error) {
	// Retrieve the agent's binary
	agentPath, err := agent.RetrieveBinary(ctx, config.GetAgentVersion(), runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}

	cmd := exec.Command(agentPath,
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
	if err := cmd.Start(); err != nil {
		return err
	}

	// Create a completely separate context <>
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-ctx.Done():
			pwi.logger.Debugf("gracefully terminating agent with PID %d", cmd.Process.Pid)
			_ = cmd.Process.Signal(os.Interrupt)
		case <-runCtx.Done():
			// agent exited
			return
		}

		const gracefulLimit = 3 * time.Second
		killPoint := time.After(gracefulLimit)

		select {
		case <-killPoint:
			pwi.logger.Debugf("killing agent with PID %d because it didn't exit after %v", cmd.Process.Pid,
				gracefulLimit)
			_ = cmd.Process.Kill()
		case <-runCtx.Done():
			// agent exited
			return
		}
	}()

	if err := cmd.Wait(); err != nil {
		return err
	}

	pwi.logger.Debugf("agent with PID %d exited normally", cmd.Process.Pid)

	return nil
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
