package worker

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	upstreampkg "github.com/cirruslabs/cirrus-cli/internal/worker/upstream"
	"github.com/getsentry/sentry-go"
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

func (worker *Worker) runTask(
	ctx context.Context,
	upstream *upstreampkg.Upstream,
	agentAwareTask *api.PollResponse_AgentAwareTask,
) {
	if _, ok := worker.tasks[agentAwareTask.TaskId]; ok {
		worker.logger.Warnf("attempted to run task %d which is already running", agentAwareTask.TaskId)
		return
	}

	taskCtx, cancel := context.WithCancel(ctx)
	worker.tasks[agentAwareTask.TaskId] = &Task{
		upstream:       upstream,
		cancel:         cancel,
		resourcesToUse: agentAwareTask.ResourcesToUse,
	}

	taskIdentification := &api.TaskIdentification{
		TaskId: agentAwareTask.TaskId,
		Secret: agentAwareTask.ClientSecret,
	}

	inst, err := persistentworker.New(agentAwareTask.Isolation, worker.logger)
	if err != nil {
		worker.logger.Errorf("failed to create an instance for the task %d: %v", agentAwareTask.TaskId, err)
		_ = upstream.TaskFailed(taskCtx, &api.TaskFailedRequest{
			TaskIdentification: taskIdentification,
			Message:            err.Error(),
		})

		return
	}

	go func() {
		localHub := sentry.CurrentHub().Clone()
		taskCtx = sentry.SetHubOnContext(taskCtx, localHub)

		defer func() {
			if err := inst.Close(); err != nil {
				worker.logger.Errorf("failed to close persistent worker instance for task %d: %v",
					agentAwareTask.TaskId, err)
			}

			worker.taskCompletions <- agentAwareTask.TaskId
		}()

		if err = upstream.TaskStarted(taskCtx, taskIdentification); err != nil {
			worker.logger.Errorf("failed to notify the server about the started task %d: %v",
				agentAwareTask.TaskId, err)

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

		if err := config.SetAgentVersionWithoutDowngrade(agentAwareTask.AgentVersion); err != nil {
			worker.logger.Warnf("failed to set agent's version for task %d: %v", agentAwareTask.TaskId, err)
		}

		err := inst.Run(taskCtx, &config)

		if err != nil && !errors.Is(err, context.Canceled) {
			worker.logger.Errorf("failed to run task %d: %v", agentAwareTask.TaskId, err)

			boundedCtx, cancel := context.WithTimeout(context.Background(), perCallTimeout)
			defer cancel()

			localHub.WithScope(func(scope *sentry.Scope) {
				scope.SetTags(cirrusSentryTags)

				buf := make([]byte, 1*1024*1024)
				n := runtime.Stack(buf, true)
				scope.SetExtra("Stack trace for all goroutines", string(buf[:n]))

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
	}()

	worker.logger.Infof("started task %d", agentAwareTask.TaskId)
}

func (worker *Worker) stopTask(taskID int64) {
	if task, ok := worker.tasks[taskID]; ok {
		task.cancel()
	}

	worker.logger.Infof("sent cancellation signal to task %d", taskID)
}

func (worker *Worker) runningTasks(upstream *upstreampkg.Upstream) (result []int64) {
	for taskID, task := range worker.tasks {
		if task.upstream != upstream {
			continue
		}

		result = append(result, taskID)
	}

	return
}

func (worker *Worker) resourcesInUse() map[string]float64 {
	result := map[string]float64{}

	for _, task := range worker.tasks {
		for key, value := range task.resourcesToUse {
			result[key] += value
		}
	}

	return result
}

func (worker *Worker) registerTaskCompletions() {
	for {
		select {
		case taskID := <-worker.taskCompletions:
			if task, ok := worker.tasks[taskID]; ok {
				task.cancel()
				delete(worker.tasks, taskID)
				worker.logger.Infof("task %d completed", taskID)
			} else {
				worker.logger.Warnf("spurious task %d completed", taskID)
			}
		default:
			return
		}
	}
}
