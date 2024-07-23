package worker

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/tart"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/vetu"
	"github.com/cirruslabs/cirrus-cli/internal/version"
	"github.com/cirruslabs/cirrus-cli/internal/worker/security"
	upstreampkg "github.com/cirruslabs/cirrus-cli/internal/worker/upstream"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
	"github.com/puzpuzpuz/xsync/v3"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"math"
	"os"
	"runtime"
	"strconv"
	"time"
)

var (
	ErrInitializationFailed = errors.New("worker initialization failed")
	ErrShutdown             = errors.New("worker is shutting down")

	tracer = otel.Tracer("worker")
	meter  = otel.Meter("worker")
)

type Worker struct {
	upstreams []*upstreampkg.Upstream

	security *security.Security

	userSpecifiedLabels    map[string]string
	userSpecifiedResources map[string]float64

	tasks           *xsync.MapOf[string, *Task]
	taskCompletions chan string

	imagesCounter     metric.Int64Counter
	tasksCounter      metric.Int64Counter
	standbyHitCounter metric.Int64Counter

	logger        logrus.FieldLogger
	echelonLogger *echelon.Logger

	standbyConfig   *StandbyConfig
	standbyInstance abstract.Instance
}

func New(opts ...Option) (*Worker, error) {
	worker := &Worker{
		upstreams: []*upstreampkg.Upstream{},

		security: security.NoSecurity(),

		userSpecifiedLabels: make(map[string]string),

		tasks:           xsync.NewMapOf[string, *Task](),
		taskCompletions: make(chan string),

		logger:        logrus.New(),
		echelonLogger: echelon.NewLogger(echelon.TraceLevel, renderers.NewSimpleRenderer(os.Stdout, nil)),
	}

	// Apply options
	for _, opt := range opts {
		opt(worker)
	}

	// Sanity check
	if len(worker.upstreams) == 0 {
		return nil, fmt.Errorf("%w: no upstreams were specified", ErrInitializationFailed)
	}

	// Image-related metrics
	imagesCounter, err := meter.Int64Counter("org.cirruslabs.persistent_worker.images.total")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to initialize images counter: %v",
			ErrInitializationFailed, err)
	}
	worker.imagesCounter = imagesCounter

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
		labels[key] = value
	}

	return &api.WorkerInfo{
		Labels:         labels,
		ResourcesTotal: worker.userSpecifiedResources,
	}
}

func (worker *Worker) Run(ctx context.Context) error {
	// Task-related metrics
	_, err := meter.Int64ObservableGauge("org.cirruslabs.persistent_worker.tasks.running_count",
		metric.WithDescription("Number of tasks running on the Persistent Worker."),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			observer.Observe(int64(worker.tasks.Size()))

			return nil
		}),
	)
	if err != nil {
		return err
	}

	worker.tasksCounter, err = meter.Int64Counter("org.cirruslabs.persistent_worker.tasks.count")
	if err != nil {
		return err
	}

	// Resource-related metrics
	_, err = meter.Float64ObservableGauge("org.cirruslabs.persistent_worker.resources.unused_count",
		metric.WithDescription("Amount of resources available for use on the Persistent Worker."),
		metric.WithFloat64Callback(func(ctx context.Context, observer metric.Float64Observer) error {
			for key, value := range worker.resourcesNotInUse() {
				observer.Observe(value, metric.WithAttributes(attribute.String("name", key)))
			}

			return nil
		}),
	)
	if err != nil {
		return err
	}

	_, err = meter.Float64ObservableGauge("org.cirruslabs.persistent_worker.resources.used_count",
		metric.WithDescription("Amount of resources used on the Persistent Worker."),
		metric.WithFloat64Callback(func(ctx context.Context, observer metric.Float64Observer) error {
			for key, value := range worker.resourcesInUse() {
				observer.Observe(value, metric.WithAttributes(attribute.String("name", key)))
			}

			return nil
		}),
	)
	if err != nil {
		return err
	}

	// standby-related metrics
	worker.standbyHitCounter, err = meter.Int64Counter("org.cirruslabs.persistent_worker.standby.hit")
	if err != nil {
		return err
	}

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
		// Create and start a standby instance if configured and not created yet
		worker.tryCreateStandby(ctx)

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

func (worker *Worker) tryCreateStandby(ctx context.Context) {
	// Do nothing if no standby instance is configured
	if worker.standbyConfig == nil {
		return
	}

	// Do nothing if the standby instance is already instantiated
	if worker.standbyInstance != nil {
		return
	}

	// Do nothing if there are tasks that are running to simplify the resource management
	if !worker.canFitResources(worker.standbyConfig.Resources) {
		return
	}

	worker.logger.Debugf("creating a new standby instance with isolation %s", worker.standbyConfig.Isolation)

	standbyInstance, err := persistentworker.New(worker.standbyConfig.Isolation, worker.security, worker.logger)
	if err != nil {
		worker.logger.Errorf("failed to create a standby instance: %v", err)

		return
	}

	worker.logger.Debugf("warming-up the standby instance")

	if err := standbyInstance.(abstract.WarmableInstance).Warmup(ctx, "standby", nil, worker.echelonLogger); err != nil {
		worker.logger.Errorf("failed to warm-up a standby instance: %v", err)

		if err := standbyInstance.Close(ctx); err != nil {
			worker.logger.Errorf("failed to terminate the standby instance after an: "+
				"unsuccessful warm-up: %v", err)
		}

		return
	}

	worker.logger.Debugf("standby instance had successfully warmed-up")

	worker.standbyInstance = standbyInstance
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
		ResourcesInUse: worker.resourcesInUse(),
	}

	request.RunningTasks, request.RunningTasksOld = FromNewTasks(worker.runningTasks(upstream))

	response, err := upstream.Poll(ctx, request)
	if err != nil {
		return err
	}

	for _, taskToStop := range ToNewTasks(response.TasksToStop, response.TasksToStopOld) {
		worker.stopTask(taskToStop)
	}

	for _, taskToStart := range response.TasksToStart {
		worker.startTask(ctx, upstream, taskToStart)
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

func ToNewTask(newTaskID string, oldTaskID int64) string {
	if newTaskID != "" {
		return newTaskID
	}

	return strconv.FormatInt(oldTaskID, 10)
}

func FromNewTasks(tasks []string) ([]string, []int64) {
	var newTaskIDs []string
	var oldTaskIDs []int64

	for _, task := range tasks {
		taskID, err := strconv.ParseInt(task, 10, 64)
		if err != nil {
			newTaskIDs = append(newTaskIDs, task)
		} else {
			oldTaskIDs = append(oldTaskIDs, taskID)
		}
	}

	return newTaskIDs, oldTaskIDs
}

func ToNewTasks(newTaskIDs []string, oldTaskIDs []int64) []string {
	result := newTaskIDs

	for _, oldTaskID := range oldTaskIDs {
		result = append(result, strconv.FormatInt(oldTaskID, 10))
	}

	return result
}
