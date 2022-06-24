package executor

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/build"
	"github.com/cirruslabs/cirrus-cli/internal/executor/build/taskstatus"
	"github.com/cirruslabs/cirrus-cli/internal/executor/endpoint"
	"github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/container"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/cirruslabs/cirrus-cli/internal/executor/pathsafe"
	"github.com/cirruslabs/cirrus-cli/internal/executor/rpc"
	"github.com/cirruslabs/cirrus-cli/internal/executor/taskfilter"
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
)

var ErrBuildFailed = errors.New("build failed")

type Executor struct {
	build *build.Build
	rpc   *rpc.RPC

	// Options
	logger                   *echelon.Logger
	taskFilter               taskfilter.TaskFilter
	baseEnvironment          map[string]string
	userSpecifiedEnvironment map[string]string
	dirtyMode                bool
	containerBackendType     string
	containerOptions         options.ContainerOptions
	tartOptions              options.TartOptions
	artifactsDir             string
}

func New(projectDir string, tasks []*api.Task, opts ...Option) (*Executor, error) {
	e := &Executor{
		taskFilter: taskfilter.MatchAnyTask(),
		baseEnvironment: environment.Merge(
			environment.Static(),
			environment.BuildID(),
			environment.ProjectSpecific(projectDir),
		),
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
	tasks, err := e.taskFilter(tasks)
	if err != nil {
		return nil, err
	}

	// Enrich task environments
	for _, task := range tasks {
		task.Environment = environment.Merge(
			// Lowest priority: common to all tasks
			e.baseEnvironment,

			// Lowest priority: task-specific
			environment.NodeInfo(task.LocalGroupId, int64(len(tasks))),
			environment.TaskInfo(task.Name, task.LocalGroupId),

			// Medium priority: task-specific
			task.Environment,

			// Highest priority: common to all tasks
			e.userSpecifiedEnvironment,
		)
	}

	// Create a build that describes what we're about to do
	b, err := build.New(projectDir, tasks, e.logger)
	if err != nil {
		return nil, err
	}
	e.build = b

	for _, task := range b.Tasks() {
		// Transform Dockerfile image names if the user provided their own template
		switch instanceWithImage := task.Instance.(type) {
		case *instance.PrebuiltInstance:
			instanceWithImage.Image, err = e.transformDockerfileImageIfNeeded(instanceWithImage.Image, true)
			if err != nil {
				return nil, err
			}
		case *container.Instance:
			instanceWithImage.Image, err = e.transformDockerfileImageIfNeeded(instanceWithImage.Image, false)
			if err != nil {
				return nil, err
			}
		}

		// Collect images that shouldn't be pulled under any circumstances
		if prebuiltInstance, ok := task.Instance.(*instance.PrebuiltInstance); ok {
			e.containerOptions.NoPullImages = append(e.containerOptions.NoPullImages, prebuiltInstance.Image)
		}

		// If not set by the user, set task's working directory based on it's instance
		task.Environment = environment.Merge(map[string]string{
			"CIRRUS_WORKING_DIR": task.Instance.WorkingDirectory(projectDir, e.dirtyMode),
		}, task.Environment)
	}

	return e, nil
}

func (e *Executor) Run(ctx context.Context) error {
	var firstErr error

	for {
		// Pick next undone task to run
		task := e.build.GetNextTask()
		if task == nil {
			break
		}

		if err := e.runSingleTask(ctx, task); err != nil {
			task.SetStatus(taskstatus.Failed)
			if firstErr == nil {
				firstErr = err
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				break
			}
		}
	}

	e.logger.Finish(firstErr == nil)
	return firstErr
}

func (e *Executor) runSingleTask(ctx context.Context, task *build.Task) error {
	rpcOpts := []rpc.Option{rpc.WithLogger(e.logger)}

	if e.artifactsDir != "" && pathsafe.IsPathSafe(task.Name) {
		taskSpecificArtifactsDir := filepath.Join(e.artifactsDir, task.Name)
		rpcOpts = append(rpcOpts, rpc.WithArtifactsDir(taskSpecificArtifactsDir))
	}

	e.rpc = rpc.New(e.build, rpcOpts...)
	if err := e.rpc.Start(ctx, "localhost:0"); err != nil {
		return err
	}
	defer e.rpc.Stop()

	e.logger.Debugf("running task %s", task.String())
	taskLogger := e.logger.Scoped(task.UniqueDescription())

	// Prepare task's instance
	instanceRunOpts := runconfig.RunConfig{
		ContainerBackendType: e.containerBackendType,
		ProjectDir:           e.build.ProjectDir,
		Endpoint:             endpoint.NewLocal(e.rpc.ContainerEndpoint(), e.rpc.DirectEndpoint()),
		ServerSecret:         e.rpc.ServerSecret(),
		ClientSecret:         e.rpc.ClientSecret(),
		TaskID:               task.ID,
		DirtyMode:            e.dirtyMode,
		ContainerOptions:     e.containerOptions,
		TartOptions:          e.tartOptions,
	}

	instanceRunOpts.SetLogger(taskLogger)

	// Respect custom agent version
	if agentVersionFromEnv, ok := task.Environment["CIRRUS_AGENT_VERSION"]; ok {
		instanceRunOpts.SetAgentVersion(agentVersionFromEnv)
	}

	// Wrap the context to enforce a timeout for this task
	ctx, cancel := context.WithTimeout(ctx, task.Timeout)

	// Run task
	if err := task.Instance.Run(ctx, &instanceRunOpts); err != nil {
		switch {
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			task.SetStatus(taskstatus.TimedOut)
		case errors.Is(err, instance.ErrUnsupportedInstance):
			task.SetStatus(taskstatus.Skipped)
		default:
			cancel()
			return err
		}
	}
	cancel()

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
	case taskstatus.Skipped:
		taskLogger.FinishWithType(echelon.FinishTypeSkipped)
		return nil
	default:
		taskLogger.Finish(false)
		return fmt.Errorf("%w: task %s %s", ErrBuildFailed, task.String(), task.Status().String())
	}

	return nil
}

func (e *Executor) transformDockerfileImageIfNeeded(reference string, strict bool) (string, error) {
	// Modify image name if the user provided a custom template
	if e.containerOptions.DockerfileImageTemplate == "" {
		return reference, nil
	}

	// Extract the already calculated hash
	const expectedMatches = 2
	re := regexp.MustCompile(`^gcr\.io/cirrus-ci-community/(.*):latest$`)
	matches := re.FindStringSubmatch(reference)
	if len(matches) != expectedMatches {
		if strict {
			return "", fmt.Errorf("%w: unknown prebuilt image format: %s", ErrBuildFailed, reference)
		}

		return reference, nil
	}
	hash := matches[1]

	// Render the template
	return strings.ReplaceAll(e.containerOptions.DockerfileImageTemplate, "%s", hash), nil
}
