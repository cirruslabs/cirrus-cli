package worker

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/tart"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/vetu"
	"github.com/cirruslabs/cirrus-cli/internal/version"
	"github.com/cirruslabs/cirrus-cli/internal/worker/security"
	upstreampkg "github.com/cirruslabs/cirrus-cli/internal/worker/upstream"
	"github.com/sirupsen/logrus"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	ErrInitializationFailed = errors.New("worker initialization failed")
	ErrShutdown             = errors.New("worker is shutting down")
)

type Worker struct {
	upstreams []*upstreampkg.Upstream

	security *security.Security

	userSpecifiedLabels    map[string]string
	userSpecifiedResources map[string]float64

	tasks           map[int64]*Task
	taskCompletions chan int64

	logger logrus.FieldLogger
}

func New(opts ...Option) (*Worker, error) {
	worker := &Worker{
		upstreams: []*upstreampkg.Upstream{},

		security: security.NoSecurity(),

		userSpecifiedLabels: make(map[string]string),

		tasks:           make(map[int64]*Task),
		taskCompletions: make(chan int64),

		logger: logrus.New(),
	}

	// Apply options
	for _, opt := range opts {
		opt(worker)
	}

	// Sanity check
	if len(worker.upstreams) == 0 {
		return nil, fmt.Errorf("%w: no upstreams were specified", ErrInitializationFailed)
	}

	return worker, nil
}

func (worker *Worker) info(workerName string) *api.WorkerInfo {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}

	const (
		ReservedLabelName         = "name"
		ReservedLabelVersion      = "version"
		ReservedLabelHostname     = "hostname"
		ReservedLabelOS           = "os"
		ReservedLabelArchitecture = "arch"
	)

	// Create base labels
	labels := map[string]string{
		ReservedLabelName:         workerName,
		ReservedLabelVersion:      version.FullVersion,
		ReservedLabelHostname:     hostname,
		ReservedLabelOS:           runtime.GOOS,
		ReservedLabelArchitecture: runtime.GOARCH,
	}

	// Merge with the user specified labels
	for key, value := range worker.userSpecifiedLabels {
		if _, alreadyExists := labels[key]; !alreadyExists {
			labels[key] = value
		}
	}

	return &api.WorkerInfo{
		Labels:         labels,
		ResourcesTotal: worker.userSpecifiedResources,
	}
}

// https://github.com/cirruslabs/cirrus-cli/issues/357
func (worker *Worker) oldWorkingDirectoryCleanup() {
	// Fix tests failing due to /tmp/cirrus-ci-build removal
	if _, runningInCi := os.LookupEnv("CIRRUS_CI"); runningInCi {
		return
	}

	tmpDir := os.TempDir()

	// Clean-up static directory[1]
	//
	//nolint:lll
	// [1]: https://github.com/cirruslabs/cirrus-ci-agent/blob/f88afe342106a6691d9e5b2d2e9187080c69fd2d/internal/executor/executor.go#L190
	staticWorkingDir := filepath.Join(tmpDir, "cirrus-ci-build")
	if err := os.RemoveAll(staticWorkingDir); err != nil {
		worker.logger.Infof("failed to clean up old cirrus-ci-build static working directory %s: %v",
			staticWorkingDir, err)
	}

	// Clean-up dynamic directories[1]
	//
	//nolint:lll
	// [1]: https://github.com/cirruslabs/cirrus-ci-agent/blob/f88afe342106a6691d9e5b2d2e9187080c69fd2d/internal/executor/executor.go#L197
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		worker.logger.Infof("failed to clean up old cirrus-task-* dynamic working directories: %v", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if strings.HasPrefix(entry.Name(), "cirrus-task-") {
			dynamicWorkingDir := filepath.Join(tmpDir, entry.Name())

			if err := os.RemoveAll(dynamicWorkingDir); err != nil {
				worker.logger.Infof("failed to clean up old cirrus-task-* dynamic working directory %s: %v",
					dynamicWorkingDir, err)
			}
		}
	}
}

func (worker *Worker) Run(ctx context.Context) error {
	// https://github.com/cirruslabs/cirrus-cli/issues/357
	worker.oldWorkingDirectoryCleanup()

	// https://github.com/cirruslabs/cirrus-cli/issues/571
	if tart.Installed() {
		if err := tart.Cleanup(); err != nil {
			worker.logger.Warnf("failed to cleanup Tart VMs: %v", err)
		}
	}

	if vetu.Installed() {
		if err := vetu.Cleanup(); err != nil {
			worker.logger.Warnf("failed to cleanup Vetu VMs: %v", err)
		}
	}

	// A sub-context to cancel out all Run() side-effects when it finishes
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		for _, upstream := range worker.upstreams {
			if err := worker.pollSingleUpstream(subCtx, upstream); err != nil {
				if errors.Is(err, ErrShutdown) {
					return nil
				}

				worker.logger.Errorf("failed to poll upstream %s: %v", upstream.Name(), err)
			}
		}

		select {
		case <-subCtx.Done():
			return nil
		case <-time.After(time.Duration(worker.pollIntervalSeconds()) * time.Second):
			// continue the loop
		}
	}
}

func (worker *Worker) pollSingleUpstream(ctx context.Context, upstream *upstreampkg.Upstream) error {
	if err := upstream.Register(ctx, worker.info(upstream.WorkerName())); err != nil {
		worker.logger.Errorf("failed to register worker with the upstream %s: %v",
			upstream.Name(), err)

		return nil
	}

	// De-register completed tasks
	worker.registerTaskCompletions()

	request := &api.PollRequest{
		WorkerInfo:     worker.info(upstream.WorkerName()),
		RunningTasks:   worker.runningTasks(upstream),
		ResourcesInUse: worker.resourcesInUse(),
	}

	response, err := upstream.Poll(ctx, request)
	if err != nil {
		return err
	}

	for _, taskToStop := range response.TasksToStop {
		worker.stopTask(taskToStop)
	}

	for _, taskToStart := range response.TasksToStart {
		worker.runTask(ctx, upstream, taskToStart)
	}

	if response.Shutdown {
		worker.logger.Infof("received shutdown signal from the upstream %s, terminating...",
			upstream.Name())

		return ErrShutdown
	}

	return nil
}

func (worker *Worker) Pause(ctx context.Context, wait bool) error {
	// A sub-context to cancel this function's side effects when it finishes
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, upstream := range worker.upstreams {
		if err := upstream.SetDisabled(subCtx, true); err != nil {
			return err
		}
	}

	if !wait {
		return nil
	}

	for _, upstream := range worker.upstreams {
		for {
			response, err := upstream.QueryRunningTasks(ctx, &api.QueryRunningTasksRequest{})
			if err != nil {
				return fmt.Errorf("upstream %s failed while waiting: %w", upstream.Name(), err)
			} else if len(response.RunningTasks) == 0 {
				// done waiting for the current upstream
				break
			}

			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Duration(upstream.PollIntervalSeconds()) * time.Second):
				// continue waiting for the current upstream
			}
		}
	}

	return nil
}

func (worker *Worker) Resume(ctx context.Context) (err error) {
	// A sub-context to cancel this function's side effects when it finishes
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, slot := range worker.upstreams {
		err = slot.SetDisabled(subCtx, false)
	}

	return
}

func (worker *Worker) pollIntervalSeconds() uint32 {
	result := uint32(math.MaxUint32)

	for _, upstream := range worker.upstreams {
		if upstream.PollIntervalSeconds() < result {
			result = upstream.PollIntervalSeconds()
		}
	}

	return result
}
