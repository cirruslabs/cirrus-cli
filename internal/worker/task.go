package worker

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/tart"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/vetu"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/worker/resources"
	upstreampkg "github.com/cirruslabs/cirrus-cli/internal/worker/upstream"
	"github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
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
	resourcesToUse resources.Resources
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

	taskIdentification := &api.TaskIdentification{
		TaskId: agentAwareTask.TaskId,
		Secret: agentAwareTask.ClientSecret,
	}

	inst, err := persistentworker.New(agentAwareTask.Isolation, worker.security, worker.logger)
	if err != nil {
		worker.logger.Errorf("failed to create an instance for the task %d: %v", agentAwareTask.TaskId, err)
		_ = upstream.TaskFailed(taskCtx, &api.TaskFailedRequest{
			TaskIdentification: taskIdentification,
			Message:            err.Error(),
		})

		return
	}

	switch typedInst := inst.(type) {
	case *tart.Tart:
		worker.imagesCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("image", typedInst.Image()),
			attribute.String("instance_type", "tart"),
		))
	case *vetu.Vetu:
		worker.imagesCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("image", typedInst.Image()),
			attribute.String("instance_type", "vetu"),
		))
	}

	go worker.runTask(taskCtx, agentAwareTask, upstream, inst, taskIdentification)

	worker.logger.Infof("started task %d", agentAwareTask.TaskId)
}

func (worker *Worker) runTask(
	ctx context.Context,
	agentAwareTask *api.PollResponse_AgentAwareTask,
	upstream *upstreampkg.Upstream,
	inst abstract.Instance,
	taskIdentification *api.TaskIdentification,
) {
	// Check if we have a standby slot that matches the task spec
	inst, err := worker.findStandby(ctx, inst, agentAwareTask.ResourcesToUse)
	if err != nil {
		worker.logger.Errorf("failed to find a standby slot: %v", err)

		return
	}

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

	// Start an OpenTelemetry span with the same attributes
	// we propagate through Sentry
	var otelAttributes []attribute.KeyValue

	for key, value := range cirrusSentryTags {
		otelAttributes = append(otelAttributes, attribute.String(key, value))
	}

	ctx, span := tracer.Start(ctx, "persistent-worker-task",
		trace.WithAttributes(otelAttributes...))
	defer span.End()

	localHub := sentry.CurrentHub().Clone()
	ctx = sentry.SetHubOnContext(ctx, localHub)

	defer func() {
		if err := inst.Close(); err != nil {
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
		TaskID:       strconv.FormatInt(agentAwareTask.TaskId, 10),
		AdditionalEnvironment: map[string]string{
			"CIRRUS_SENTRY_TAGS": strings.Join(cirrusSentryTagsFormatted, ","),
		},
	}

	if err := config.SetAgentVersionWithoutDowngrade(agentAwareTask.AgentVersion); err != nil {
		worker.logger.Warnf("failed to set agent's version for task %d: %v", agentAwareTask.TaskId, err)
	}

	err = inst.Run(ctx, &config)

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

func (worker *Worker) findStandby(
	ctx context.Context,
	inst abstract.Instance,
	resourcesToUse resources.Resources,
) (abstract.Instance, error) {
	// Do nothing if standby is not supported by this instance type
	standbyCapableInstance, ok := inst.(abstract.StandbyCapableInstance)
	if !ok {
		return inst, nil
	}

	// Pull the instance and figure out its FQN
	if err := standbyCapableInstance.Pull(ctx, nil, nil); err != nil {
		return nil, err
	}

	fqn, err := standbyCapableInstance.FQN(ctx)
	if err != nil {
		return nil, err
	}

	// Find a standby slot that matches both the FQN and the isolation specification
	if standbyInstance := worker.standby.FindSlot(fqn, standbyCapableInstance.Isolation()); standbyInstance != nil {
		if err := standbyCapableInstance.Close(); err != nil {
			panic(err)
		}

		return standbyInstance, nil
	}

	// Kill off standby instances if no appropriate standby
	// instance is found and we have a resource shortage
	if !worker.resourcesNotInUse().CanFit(resourcesToUse) {
		needResources := resourcesToUse.Sub(worker.resourcesNotInUse())

		worker.standby.StopSlots(needResources)
	}

	return inst, nil
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

func (worker *Worker) resourcesNotInUse() resources.Resources {
	result := maps.Clone(worker.userSpecifiedResources)

	worker.tasks.Range(func(taskID int64, task *Task) bool {
		for key, value := range task.resourcesToUse {
			result[key] -= value
		}
		return true
	})

	return result
}

func (worker *Worker) resourcesInUse() resources.Resources {
	result := resources.New()

	worker.tasks.Range(func(taskID int64, task *Task) bool {
		result = result.Add(task.resourcesToUse)

		return true
	})

	return result
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
