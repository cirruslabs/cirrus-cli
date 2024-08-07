package worker

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	upstreampkg "github.com/cirruslabs/cirrus-cli/internal/worker/upstream"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"maps"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"
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
	if _, ok := worker.tasks.Load(agentAwareTask.TaskId); ok {
		worker.logger.Warnf("attempted to run task %d which is already running", agentAwareTask.TaskId)
		return
	}

	taskCtx, cancel := context.WithCancel(ctx)
	worker.tasks.Store(agentAwareTask.TaskId, &Task{
		upstream:       upstream,
		cancel:         cancel,
		resourcesToUse: agentAwareTask.ResourcesToUse,
	})
	worker.tasksCounter.Add(ctx, 1)

	taskIdentification := &api.TaskIdentification{
		TaskId: agentAwareTask.TaskId,
		Secret: agentAwareTask.ClientSecret,
	}

	inst, err := worker.getInstance(ctx, agentAwareTask.Isolation, agentAwareTask.ResourcesToUse)
	if err != nil {
		worker.logger.Errorf("failed to create an instance for the task %d: %v", agentAwareTask.TaskId, err)
		_ = upstream.TaskFailed(taskCtx, &api.TaskFailedRequest{
			TaskIdentification: taskIdentification,
			Message:            err.Error(),
		})

		return
	}

	worker.imagesCounter.Add(ctx, 1, metric.WithAttributes(inst.Attributes()...))
	go worker.runTask(taskCtx, agentAwareTask, upstream, inst, taskIdentification)

	worker.logger.Infof("started task %d", agentAwareTask.TaskId)
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

		// Return the standby instance if matches the isolation required by the task
		if proto.Equal(worker.standbyConfig.Isolation, isolation) {
			worker.logger.Debugf("standby instance matches the task's isolation configuration, " +
				"yielding it to the task")
			worker.standbyHitCounter.Add(ctx, 1)

			return standbyInstance, nil
		}

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
		worker.resourceModifierManager.Acquire(resourcesToUse), worker.logger)
}

func (worker *Worker) runTask(
	ctx context.Context,
	agentAwareTask *api.PollResponse_AgentAwareTask,
	upstream *upstreampkg.Upstream,
	inst abstract.Instance,
	taskIdentification *api.TaskIdentification,
) {
	// Provide tags for Sentry: task ID and upstream worker name
	cirrusSentryTags := map[string]string{
		"cirrus.task_id":              strconv.FormatInt(agentAwareTask.TaskId, 10),
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

	localHub := sentry.CurrentHub().Clone()
	ctx = sentry.SetHubOnContext(ctx, localHub)

	defer func() {
		if err := inst.Close(ctx); err != nil {
			worker.logger.Errorf("failed to close persistent worker instance for task %d: %v",
				agentAwareTask.TaskId, err)
		}

		worker.taskCompletions <- agentAwareTask.TaskId
	}()

	if err := upstream.TaskStarted(ctx, taskIdentification); err != nil {
		worker.logger.Errorf("failed to notify the server about the started task %d: %v",
			agentAwareTask.TaskId, err)

		return
	}

	var cirrusSentryTagsFormatted []string
	for k, v := range cirrusSentryTags {
		cirrusSentryTagsFormatted = append(cirrusSentryTagsFormatted, fmt.Sprintf("%s=%s", k, v))
	}

	config := runconfig.RunConfig{
		ProjectDir:   "",
		Endpoint:     upstream.AgentEndpoint(),
		ServerSecret: agentAwareTask.ServerSecret,
		ClientSecret: agentAwareTask.ClientSecret,
		TaskID:       agentAwareTask.TaskId,
		AdditionalEnvironment: map[string]string{
			"CIRRUS_SENTRY_TAGS": strings.Join(cirrusSentryTagsFormatted, ","),
		},
	}

	if err := config.SetCLIVersionWithoutDowngrade(agentAwareTask.CliVersion); err != nil {
		worker.logger.Warnf("failed to set CLI's version for task %d: %v", agentAwareTask.TaskId, err)
	}

	err := inst.Run(ctx, &config)

	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(ctx.Err(), context.Canceled) {
		worker.logger.Errorf("failed to run task %d: %v", agentAwareTask.TaskId, err)

		boundedCtx, cancel := context.WithTimeout(context.Background(), perCallTimeout)
		defer cancel()

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
			TaskIdentification: taskIdentification,
			Message:            err.Error(),
		})
		if err != nil {
			worker.logger.Errorf("failed to notify the server about the failed task %d: %v",
				agentAwareTask.TaskId, err)
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

	boundedCtx, cancel := context.WithTimeout(context.Background(), perCallTimeout)
	defer cancel()

	if err = upstream.TaskStopped(boundedCtx, taskIdentification); err != nil {
		worker.logger.Errorf("failed to notify the server about the stopped task %d: %v",
			agentAwareTask.TaskId, err)
		localHub.WithScope(func(scope *sentry.Scope) {
			scope.SetTags(cirrusSentryTags)
			scope.SetLevel(sentry.LevelFatal)
			localHub.CaptureMessage(fmt.Sprintf("failed to notify the server about the stopped task: %v", err))
		})
		return
	}
}

func (worker *Worker) stopTask(taskID int64) {
	if task, ok := worker.tasks.Load(taskID); ok {
		task.cancel()
	}

	worker.logger.Infof("sent cancellation signal to task %d", taskID)
}

func (worker *Worker) runningTasks(upstream *upstreampkg.Upstream) (result []int64) {
	worker.tasks.Range(func(taskID int64, task *Task) bool {
		if task.upstream == upstream {
			result = append(result, taskID)
		}
		return true
	})

	return
}

func (worker *Worker) resourcesNotInUse() map[string]float64 {
	result := maps.Clone(worker.userSpecifiedResources)

	worker.tasks.Range(func(taskID int64, task *Task) bool {
		for key, value := range task.resourcesToUse {
			result[key] -= value
		}
		return true
	})

	return result
}

func (worker *Worker) resourcesInUse() map[string]float64 {
	result := map[string]float64{}

	worker.tasks.Range(func(taskID int64, task *Task) bool {
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
				worker.logger.Infof("task %d completed", taskID)
			} else {
				worker.logger.Warnf("spurious task %d completed", taskID)
			}
		default:
			return
		}
	}
}
