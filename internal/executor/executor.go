package executor

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/build"
	"github.com/cirruslabs/cirrus-cli/internal/executor/build/taskstatus"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"github.com/cirruslabs/cirrus-cli/internal/executor/rpc"
	"github.com/cirruslabs/cirrus-cli/internal/executor/taskfilter"
	"github.com/sirupsen/logrus"
	"io/ioutil"
)

var ErrBuildFailed = errors.New("build failed")

type Executor struct {
	build *build.Build
	rpc   *rpc.RPC

	logger     *logrus.Logger
	taskFilter taskfilter.TaskFilter
}

func New(projectDir string, tasks []*api.Task, opts ...Option) (*Executor, error) {
	e := &Executor{
		taskFilter: taskfilter.MatchAnyTask(),
	}

	// Apply options
	for _, opt := range opts {
		opt(e)
	}

	// Apply default options (to cover those that weren't specified)
	if e.logger == nil {
		e.logger = logrus.New()
		e.logger.Out = ioutil.Discard
	}

	// Filter tasks (e.g. if a user wants to run only a specific task without dependencies)
	tasks = e.taskFilter(tasks)

	// Create a build that describes what we're about to do
	b, err := build.New(projectDir, tasks)
	if err != nil {
		return nil, err
	}

	e.build = b
	e.rpc = rpc.New(b, rpc.WithLogger(e.logger))

	return e, nil
}

func (e *Executor) Run(ctx context.Context) error {
	e.rpc.Start()
	defer e.rpc.Stop()

	for {
		// Pick next undone task to run
		task := e.build.GetNextTask()
		if task == nil {
			break
		}

		e.logger.Infof("running task %s", task.String())

		// Prepare task's instance
		taskInstance := task.Instance
		instanceRunOpts := instance.RunConfig{
			ProjectDir:   e.build.ProjectDir,
			Endpoint:     e.rpc.Endpoint(),
			ServerSecret: e.rpc.ServerSecret(),
			ClientSecret: e.rpc.ClientSecret(),
			TaskID:       task.ID,
			Logger:       e.logger,
		}

		// Wrap the context to enforce a timeout for this task
		ctx, cancel := context.WithTimeout(ctx, task.Timeout)

		// Run task
		var timedOut bool
		if err := taskInstance.Run(ctx, &instanceRunOpts); err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				timedOut = true
			} else {
				cancel()
				return err
			}
		}
		cancel()

		// Handle timeout
		if timedOut {
			task.SetStatus(taskstatus.TimedOut)
		}

		e.logger.Infof("task %s %s", task.String(), task.Status().String())

		// Bail-out if the task has failed
		if task.Status() != taskstatus.Succeeded {
			return fmt.Errorf("%w: task %s %s", ErrBuildFailed, task.String(), task.Status().String())
		}
	}

	return nil
}
