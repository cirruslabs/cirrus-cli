package worker

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/chacha/pkg/localnetworkhelper"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/tart"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/vetu"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/version"
	"github.com/cirruslabs/cirrus-cli/internal/worker/chacha"
	"github.com/cirruslabs/cirrus-cli/internal/worker/resourcemodifier"
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
	"google.golang.org/protobuf/proto"
	"math"
	"os"
	"runtime"
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

	security                *security.Security
	resourceModifierManager *resourcemodifier.Manager

	userSpecifiedLabels    map[string]string
	userSpecifiedResources map[string]float64

	tasks           *xsync.MapOf[string, *Task]
	taskCompletions chan string

	imagesCounter               metric.Int64Counter
	tasksCounter                metric.Int64Counter
	taskExecutionTimeHistogram  metric.Float64Histogram
	standbyHitCounter           metric.Int64Counter
	standbyMissCounter          metric.Int64Counter
	standbyInstanceAgeHistogram metric.Float64Histogram

	logger        logrus.FieldLogger
	echelonLogger *echelon.Logger

	standbyParameters        *api.StandbyInstanceParameters
	standbyInstance          abstract.Instance
	standbyInstanceStartedAt time.Time

	tartPrePull        *TartPrePull
	chacha             *chacha.Chacha
	localNetworkHelper *localnetworkhelper.LocalNetworkHelper
}

func New(opts ...Option) (*Worker, error) {
	worker := &Worker{
		upstreams: []*upstreampkg.Upstream{},

		security:                security.NoSecurity(),
		resourceModifierManager: resourcemodifier.NewManager(),

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

	worker.taskExecutionTimeHistogram, err = meter.Float64Histogram(
		"org.cirruslabs.persistent_worker.tasks.execution_time",
		metric.WithDescription("Task execution time."),
		metric.WithUnit("s"),
	)
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
	worker.standbyMissCounter, err = meter.Int64Counter("org.cirruslabs.persistent_worker.standby.miss")
	if err != nil {
		return err
	}

	worker.standbyInstanceAgeHistogram, err = meter.Float64Histogram(
		"org.cirruslabs.persistent_worker.standby.age",
		metric.WithDescription("Standby instance age at the moment of relinquishing the ownership."),
		metric.WithUnit("s"),
	)
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
		if err := worker.pollUpstreams(subCtx); err != nil {
			if errors.Is(err, ErrShutdown) {
				return nil
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

func (worker *Worker) Close() error {
	if worker.chacha != nil {
		return worker.chacha.Close()
	}

	return nil
}

func (worker *Worker) tryCreateStandby(ctx context.Context) error {
	// Do nothing if no standby instance is configured
	if worker.standbyParameters == nil {
		return nil
	}

	// Do nothing if the standby instance is already instantiated
	if worker.standbyInstance != nil {
		return nil
	}

	// Do nothing if there are tasks that are running to simplify the resource management
	if !worker.canFitResources(worker.standbyParameters.Resources) {
		return nil
	}

	worker.logger.Debugf("creating a new standby instance with isolation %s", worker.standbyParameters.Isolation)

	standbyInstance, err := persistentworker.New(worker.standbyParameters.Isolation, worker.security,
		worker.resourceModifierManager.Acquire(worker.standbyParameters.Resources), worker.logger)
	if err != nil {
		worker.logger.Errorf("failed to create a standby instance: %v", err)

		return err
	}

	lazyPull := false

	if worker.tartPrePull != nil {
		// Pre-pull the configured Tart VM images first
		for _, image := range worker.tartPrePull.Images {
			for _, attr := range standbyInstance.Attributes() {
				if attr.Key == "image" && attr.Value.AsString() == image {
					lazyPull = true
				}
			}

			if !worker.tartPrePull.NeedsPrePull() {
				continue
			}

			worker.logger.Infof("pre-pulling Tart VM image %q...", image)

			if err := tart.PrePull(ctx, image, worker.echelonLogger); err != nil {
				worker.logger.Errorf("failed to pre-pull Tart VM image %q: %v", image, err)
				continue
			}
		}
		worker.tartPrePull.LastCheck = time.Now()
	}

	worker.logger.Debugf("warming-up the standby instance")

	runConfig := &runconfig.RunConfig{
		Chacha:             worker.chacha,
		LocalNetworkHelper: worker.localNetworkHelper,
	}

	if err := standbyInstance.(abstract.WarmableInstance).Warmup(ctx, "standby", nil, lazyPull,
		worker.standbyParameters.Warmup, runConfig, worker.echelonLogger); err != nil {
		worker.logger.Errorf("failed to warm-up a standby instance: %v", err)

		if cleanupErr := standbyInstance.Close(ctx); cleanupErr != nil {
			worker.logger.Errorf("failed to terminate the standby instance after an "+
				"unsuccessful warm-up: %v", cleanupErr)
		}

		return err
	}

	worker.logger.Debugf("standby instance had successfully warmed-up")

	worker.standbyInstance = standbyInstance
	worker.standbyInstanceStartedAt = time.Now()

	return nil
}

func (worker *Worker) pollUpstreams(ctx context.Context) error {
	// Create and start a standby instance if configured and not created yet
	standbyErr := worker.tryCreateStandby(ctx)
	if standbyErr != nil {
		worker.logger.Error("failed to create a standby instance... backing off...")
		return standbyErr
	}

	for _, upstream := range worker.upstreams {
		if pollErr := worker.pollSingleUpstream(ctx, upstream); pollErr != nil {
			worker.logger.Errorf("failed to poll the upstream %s: %v", upstream.Name(), pollErr)
			return pollErr
		}
	}

	return nil
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
		WorkerInfo:      worker.info(upstream.WorkerName()),
		RunningTasks:    worker.runningTasks(upstream),
		OldRunningTasks: worker.oldRunningTasks(upstream),
		ResourcesInUse:  worker.resourcesInUse(),
	}

	if worker.standbyInstance != nil {
		request.AvailableStandbyInstancesInformation = []*api.StandbyInstanceInformation{
			{
				Parameters: worker.standbyParameters,
				AgeSeconds: uint64(time.Since(worker.standbyInstanceStartedAt).Seconds()),
			},
		}
	}

	response, err := upstream.Poll(ctx, request)
	if err != nil {
		return err
	}

	for _, taskToStop := range response.TasksToStop {
		worker.stopTask(taskToStop)
	}

	for _, taskToStop := range response.OldTaskIdsToStop {
		worker.stopTask(fmt.Sprintf("%d", taskToStop))
	}

	for _, taskToStart := range response.TasksToStart {
		worker.startTask(ctx, upstream, taskToStart)
	}

	if response.Shutdown {
		worker.logger.Infof("received shutdown signal from the upstream %s, terminating...",
			upstream.Name())

		return ErrShutdown
	}

	if len(response.UpdatedStandbyInstances) > 0 {
		worker.UpdateStandby(ctx, response.UpdatedStandbyInstances[0])
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
			} else if len(response.RunningTasks) == 0 && len(response.OldRunningTasks) == 0 {
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

func (worker *Worker) Resume(ctx context.Context) error {
	// A sub-context to cancel this function's side effects when it finishes
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, slot := range worker.upstreams {
		if err := slot.SetDisabled(subCtx, false); err != nil {
			return err
		}
	}

	return nil
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

func (worker *Worker) UpdateStandby(ctx context.Context, parameters *api.StandbyInstanceParameters) {
	if worker.standbyInstance != nil && !proto.Equal(worker.standbyParameters, parameters) {
		worker.logger.Infof("terminating the standby instance since the parameters have changed")

		if err := worker.standbyInstance.Close(ctx); err != nil {
			worker.logger.Errorf("failed to terminate the standby instance: %v", err)
		}

		worker.standbyInstance = nil
		worker.standbyInstanceStartedAt = time.Time{}
	}

	worker.standbyParameters = parameters
}
