package worker

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	upstreampkg "github.com/cirruslabs/cirrus-cli/internal/worker/upstream"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

const perCallTimeout = 15 * time.Second

type Task struct {
	upstream       *upstreampkg.Upstream
	cancel         context.CancelFunc
	resourcesToUse map[string]float64
}

func (worker *Worker) startTask(
	ctx context.Context,
	upstream *upstreampkg.Upstream,
	agentAwareTask *api.PollResponse_AgentAwareTask,
) {
	// Propagate task's W3C Trace Context to a worker
	ctx = propagation.TraceContext{}.Extract(ctx, propagation.MapCarrier{
		"traceparent": agentAwareTask.Traceparent,
		"tracestate":  agentAwareTask.Tracestate,
	})

	taskID := agentAwareTask.TaskId
	if taskID == "" {
		taskID = fmt.Sprintf("%d", agentAwareTask.OldTaskId)
	}
	if _, ok := worker.tasks.Load(taskID); ok {
		worker.logger.Warnf("attempted to run task %s which is already running", taskID)
		return
	}

	ctx = metadata.AppendToOutgoingContext(ctx,
		"org.cirruslabs.task-id", taskID,
		"org.cirruslabs.client-secret", agentAwareTask.ClientSecret,
	)

	taskCtx, cancel := context.WithCancel(ctx)

	worker.tasks.Store(taskID, &Task{
		upstream:       upstream,
		cancel:         cancel,
		resourcesToUse: agentAwareTask.ResourcesToUse,
	})
	worker.tasksCounter.Add(ctx, 1)

	inst, err := worker.getInstance(taskCtx, agentAwareTask.Isolation, agentAwareTask.ResourcesToUse)
	if err != nil {
		worker.logger.Errorf("failed to create an instance for the task %s: %v", taskID, err)
		_ = upstream.TaskFailed(taskCtx, &api.TaskFailedRequest{
			TaskIdentification: api.OldTaskIdentification(taskID, agentAwareTask.ClientSecret),
			Message:            err.Error(),
		})

		return
	}

	worker.imagesCounter.Add(ctx, 1, metric.WithAttributes(inst.Attributes()...))
	go worker.runTask(taskCtx, upstream, inst, agentAwareTask.CliVersion,
		taskID, agentAwareTask.ClientSecret, agentAwareTask.ServerSecret)

	worker.logger.Infof("started task %s", taskID)
}

func (worker *Worker) getInstance(
	ctx context.Context,
	isolation *api.Isolation,
	resourcesToUse map[string]float64,
) (abstract.Instance, error) {
	if standbyInstance := worker.standbyInstance; standbyInstance != nil {
		// Relinquish our ownership of the standby instance since
		// we'll either return it to the task or terminate it
		worker.standbyInstance = nil
		worker.standbyInstanceStartedAt = time.Time{}

		// Return the standby instance if matches the isolation required by the task
		if proto.Equal(worker.standbyParameters.Isolation, isolation) {
			worker.logger.Debugf("standby instance matches the task's isolation configuration, " +
				"yielding it to the task")
			worker.standbyHitCounter.Add(ctx, 1, metric.WithAttributes(standbyInstance.Attributes()...))

			return standbyInstance, nil
		}
		worker.standbyMissCounter.Add(ctx, 1, metric.WithAttributes(standbyInstance.Attributes()...))

		// Otherwise terminate the standby instance to simplify the resource management
		worker.logger.Debugf("standby instance does not match the task's isolation configuration, " +
			"terminating it")

		if err := standbyInstance.Close(ctx); err != nil {
			worker.logger.Errorf("failed to terminate the standby instance: %v", err)
		} else {
			worker.logger.Debugf("standby instance had successfully terminated")
		}
	}

	// Otherwise proceed with creating a new instance
	return persistentworker.New(isolation, worker.security,
		worker.resourceModifierManager.Acquire(resourcesToUse), worker.tuning, worker.logger)
}

func (worker *Worker) runTask(
	ctx context.Context,
	upstream *upstreampkg.Upstream,
	inst abstract.Instance,
	cliVersion string,
	taskID string,
	clientSecret string,
	serverSecret string,
) {
	// Provide tags for Sentry: task ID and upstream worker name
	cirrusSentryTags := map[string]string{
		"cirrus.task_id":              taskID,
		"cirrus.upstream_worker_name": upstream.WorkerName(),
	}

	// Provide tags for Sentry: upstream hostname
	if url, err := url.Parse(upstream.Name()); err == nil {
		cirrusSentryTags["cirrus.upstream_hostname"] = url.Host
	} else {
		cirrusSentryTags["cirrus.upstream_hostname"] = upstream.Name()
	}

	var otelAttributes = inst.Attributes()

	// add Sentry tags to OpenTelemetry attributes
	for key, value := range cirrusSentryTags {
		otelAttributes = append(otelAttributes, attribute.String(key, value))
	}

	ctx, span := tracer.Start(ctx, "persistent-worker-task",
		trace.WithAttributes(otelAttributes...))
	defer span.End()

	backgroundCtxWithSpan := trace.ContextWithSpan(context.Background(), trace.SpanFromContext(ctx))

	localHub := sentry.CurrentHub().Clone()
	ctx = sentry.SetHubOnContext(ctx, localHub)

	defer func() {
		if err := inst.Close(ctx); err != nil {
			worker.logger.Errorf("failed to close persistent worker instance for task %s: %v",
				taskID, err)
		}

		worker.taskCompletions <- taskID
	}()

	if err := upstream.TaskStarted(ctx, api.OldTaskIdentification(taskID, clientSecret)); err != nil {
		worker.logger.Errorf("failed to notify the server about the started task %s: %v",
			taskID, err)

		return
	}

	var cirrusSentryTagsFormatted []string
	for k, v := range cirrusSentryTags {
		cirrusSentryTagsFormatted = append(cirrusSentryTagsFormatted, fmt.Sprintf("%s=%s", k, v))
	}

	mapCarrier := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(ctx, mapCarrier)

	config := runconfig.RunConfig{
		ProjectDir:   "",
		Endpoint:     upstream.AgentEndpoint(),
		ServerSecret: serverSecret,
		ClientSecret: clientSecret,
		TaskID:       taskID,
		AdditionalEnvironment: map[string]string{
			"CIRRUS_SENTRY_TAGS": strings.Join(cirrusSentryTagsFormatted, ","),
			// Propagate task's W3C Trace Context to an agent using the environment variables[1]
			//
			// [1]: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/context/env-carriers.md
			"TRACEPARENT": mapCarrier.Get("traceparent"),
			"TRACESTATE":  mapCarrier.Get("tracestate"),
		},
		Chacha:             worker.chacha,
		LocalNetworkHelper: worker.localNetworkHelper,
	}

	if worker.chacha != nil {
		maps.Copy(config.AdditionalEnvironment, worker.chacha.AgentEnvironmentVariables())
	}

	if err := config.SetCLIVersionWithoutDowngrade(cliVersion); err != nil {
		worker.logger.Warnf("failed to set CLI's version for task %s: %v", taskID, err)
	}

	err := inst.Run(ctx, &config)
	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(ctx.Err(), context.Canceled) {
		worker.logger.Errorf("failed to run task %s: %v", taskID, err)

		boundedCtx, cancel := context.WithTimeout(backgroundCtxWithSpan, perCallTimeout)
		defer cancel()

		if md, ok := metadata.FromOutgoingContext(ctx); ok {
			boundedCtx = metadata.NewOutgoingContext(boundedCtx, md)
		}

		localHub.WithScope(func(scope *sentry.Scope) {
			scope.SetTags(cirrusSentryTags)

			buf := make([]byte, 1*1024*1024)
			n := runtime.Stack(buf, true)
			scope.AddAttachment(&sentry.Attachment{
				Filename:    "goroutines-stacktrace.txt",
				ContentType: "text/plain",
				Payload:     buf[:n],
			})

			localHub.CaptureException(err)
		})

		err := upstream.TaskFailed(boundedCtx, &api.TaskFailedRequest{
			TaskIdentification: api.OldTaskIdentification(taskID, clientSecret),
			Message:            err.Error(),
		})
		if err != nil {
			worker.logger.Errorf("failed to notify the server about the failed task %s: %v",
				taskID, err)
			localHub.WithScope(func(scope *sentry.Scope) {
				scope.SetTags(cirrusSentryTags)

				buf := make([]byte, 1*1024*1024)
				n := runtime.Stack(buf, true)
				scope.AddAttachment(&sentry.Attachment{
					Filename:    "goroutines-stacktrace.txt",
					ContentType: "text/plain",
					Payload:     buf[:n],
				})

				scope.SetLevel(sentry.LevelFatal)
				localHub.CaptureMessage(fmt.Sprintf("failed to notify the server about the failed task: %v", err))
			})
		}
	}

	boundedCtx, cancel := context.WithTimeout(backgroundCtxWithSpan, perCallTimeout)
	defer cancel()

	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		boundedCtx = metadata.NewOutgoingContext(boundedCtx, md)
	}

	if err = upstream.TaskStopped(boundedCtx, api.OldTaskIdentification(taskID, clientSecret)); err != nil {
		worker.logger.Errorf("failed to notify the server about the stopped task %s: %v",
			taskID, err)
		localHub.WithScope(func(scope *sentry.Scope) {
			scope.SetTags(cirrusSentryTags)
			scope.SetLevel(sentry.LevelFatal)
			localHub.CaptureMessage(fmt.Sprintf("failed to notify the server about the stopped task: %v", err))
		})
		return
	}
}

func (worker *Worker) stopTask(taskID string) {
	if task, ok := worker.tasks.Load(taskID); ok {
		task.cancel()
	}

	worker.logger.Infof("sent cancellation signal to task %s", taskID)
}

func (worker *Worker) oldRunningTasks(upstream *upstreampkg.Upstream) (result []int64) {
	for _, taskID := range worker.runningTasks(upstream) {
		if taskIDInt, err := strconv.ParseInt(taskID, 10, 64); err == nil {
			result = append(result, taskIDInt)
		}
	}

	return
}

func (worker *Worker) runningTasks(upstream *upstreampkg.Upstream) (result []string) {
	worker.tasks.Range(func(taskID string, task *Task) bool {
		if task.upstream == upstream {
			result = append(result, taskID)
		}
		return true
	})

	return
}

func (worker *Worker) resourcesNotInUse() map[string]float64 {
	result := maps.Clone(worker.userSpecifiedResources)

	worker.tasks.Range(func(taskID string, task *Task) bool {
		for key, value := range task.resourcesToUse {
			result[key] -= value
		}
		return true
	})

	return result
}

func (worker *Worker) resourcesInUse() map[string]float64 {
	result := map[string]float64{}

	worker.tasks.Range(func(taskID string, task *Task) bool {
		for key, value := range task.resourcesToUse {
			result[key] += value
		}
		return true
	})

	return result
}

func (worker *Worker) canFitResources(resources map[string]float64) bool {
	resourcesNotInUse := worker.resourcesNotInUse()
	for key, value := range resources {
		if resourcesNotInUse[key] < value {
			return false
		}
	}
	return true
}

func (worker *Worker) registerTaskCompletions() {
	for {
		select {
		case taskID := <-worker.taskCompletions:
			if task, ok := worker.tasks.Load(taskID); ok {
				task.cancel()
				worker.tasks.Delete(taskID)
				worker.logger.Infof("task %s completed", taskID)
			} else {
				worker.logger.Warnf("spurious task %s completed", taskID)
			}
		default:
			return
		}
	}
}
