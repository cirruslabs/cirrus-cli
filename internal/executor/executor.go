package executor

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/build"
	"github.com/cirruslabs/cirrus-cli/internal/executor/build/taskstatus"
	"github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/cirruslabs/cirrus-cli/internal/executor/rpc"
	"github.com/cirruslabs/cirrus-cli/internal/executor/taskfilter"
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
	"io/ioutil"
)

var ErrBuildFailed = errors.New("build failed")

type Executor struct {
	build *build.Build
	rpc   *rpc.RPC

	// Options
	logger                   *echelon.Logger
	taskFilter               taskfilter.TaskFilter
	userSpecifiedEnvironment map[string]string
	dirtyMode                bool
	dockerOptions            options.DockerOptions
}

func New(projectDir string, tasks []*api.Task, opts ...Option) (*Executor, error) {
	e := &Executor{
		taskFilter:               taskfilter.MatchAnyTask(),
		userSpecifiedEnvironment: make(map[string]string),
	}

	// Apply options
	for _, opt := range opts {
		opt(e)
	}

	// Apply default options (to cover those that weren't specified)
	if e.logger == nil {
		renderer := renderers.NewSimpleRenderer(ioutil.Discard, nil)
		e.logger = echelon.NewLogger(echelon.InfoLevel, renderer)
	}

	// Filter tasks (e.g. if a user wants to run only a specific task without dependencies)
	tasks = e.taskFilter(tasks)

	// Enrich task environments
	commonToAllTasks := environment.Merge(environment.Static(), environment.BuildID())
	for _, task := range tasks {
		task.Environment = environment.Merge(
			// Lowest priority
			commonToAllTasks,
			environment.NodeInfo(task.LocalGroupId, int64(len(tasks))),
			environment.TaskInfo(task.Name, task.LocalGroupId),

			// Medium priority
			task.Environment,

			// Highest priority
			e.userSpecifiedEnvironment,
		)
	}

	// Create a build that describes what we're about to do
	b, err := build.New(projectDir, tasks)
	if err != nil {
		return nil, err
	}

	e.build = b
	e.rpc = rpc.New(b, rpc.WithLogger(e.logger))

	// Collect images that shouldn't be pulled under any circumstances
	for _, task := range b.Tasks() {
		prebuiltInstance, ok := task.Instance.(*instance.PrebuiltInstance)
		if !ok {
			continue
		}

		e.dockerOptions.NoPullImages = append(e.dockerOptions.NoPullImages, prebuiltInstance.Image)
	}

	return e, nil
}

func (e *Executor) Run(ctx context.Context) error {
	if err := e.rpc.Start(ctx); err != nil {
		return err
	}
	defer e.rpc.Stop()

	for {
		// Pick next undone task to run
		task := e.build.GetNextTask()
		if task == nil {
			break
		}

		e.logger.Debugf("running task %s", task.String())
		taskLogger := e.logger.Scoped(task.UniqueName())

		// Prepare task's instance
		taskInstance := task.Instance
		instanceRunOpts := instance.RunConfig{
			ProjectDir:    e.build.ProjectDir,
			Endpoint:      e.rpc.Endpoint(),
			ServerSecret:  e.rpc.ServerSecret(),
			ClientSecret:  e.rpc.ClientSecret(),
			TaskID:        task.ID,
			Logger:        taskLogger,
			DirtyMode:     e.dirtyMode,
			DockerOptions: e.dockerOptions,
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

		// Handle prebuilt instance which doesn't require any tasks to be run to be considered successful
		_, isPrebuilt := task.Instance.(*instance.PrebuiltInstance)
		if isPrebuilt && task.Status() == taskstatus.New {
			task.SetStatus(taskstatus.Succeeded)
		}

		switch task.Status() {
		case taskstatus.Succeeded:
			e.logger.Debugf("task %s %s", task.String(), task.Status().String())
			taskLogger.Finish(true)
		case taskstatus.New:
			taskLogger.Finish(false)
			return fmt.Errorf("%w: instance terminated before the task %s had a chance to run", ErrBuildFailed, task.String())
		default:
			taskLogger.Finish(false)
			return fmt.Errorf("%w: task %s %s", ErrBuildFailed, task.String(), task.Status().String())
		}

		// Bail-out if the task has failed
		if task.Status() != taskstatus.Succeeded {
			taskLogger.Finish(false)
			return fmt.Errorf("%w: task %s %s", ErrBuildFailed, task.String(), task.Status().String())
		}
	}

	return nil
}
