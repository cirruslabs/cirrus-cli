package worker

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"google.golang.org/grpc"
)

func (worker *Worker) runTask(ctx context.Context, agentAwareTask *api.PollResponse_AgentAwareTask) {
	if _, ok := worker.tasks[agentAwareTask.TaskId]; ok {
		worker.logger.Warnf("attempted to run task %d which is already running", agentAwareTask.TaskId)
		return
	}

	taskCtx, cancel := context.WithCancel(ctx)
	worker.tasks[agentAwareTask.TaskId] = cancel

	inst, err := instance.NewPersistentWorkerInstance()
	if err != nil {
		worker.logger.Errorf("failed to create an instance for the task %d: %v", agentAwareTask.TaskId, err)
		return
	}

	go func() {
		defer func() {
			worker.taskCompletions <- agentAwareTask.TaskId
		}()

		taskIdentification := &api.TaskIdentification{
			TaskId: agentAwareTask.TaskId,
			Secret: agentAwareTask.ClientSecret,
		}
		_, err = worker.rpcClient.TaskStarted(taskCtx, taskIdentification, grpc.PerRPCCredentials(worker))
		if err != nil {
			worker.logger.Errorf("failed to notify the server about the started task %d: %v",
				agentAwareTask.TaskId, err)
			return
		}

		if err := inst.Run(taskCtx, &instance.RunConfig{
			ProjectDir:        "",
			ContainerEndpoint: worker.rpcEndpoint,
			DirectEndpoint:    worker.rpcEndpoint,
			ServerSecret:      agentAwareTask.ServerSecret,
			ClientSecret:      agentAwareTask.ClientSecret,
			TaskID:            agentAwareTask.TaskId,
		}); err != nil {
			worker.logger.Errorf("failed to run task %d: %v", agentAwareTask.TaskId, err)
		}

		_, err = worker.rpcClient.TaskStopped(taskCtx, taskIdentification, grpc.PerRPCCredentials(worker))
		if err != nil {
			worker.logger.Errorf("failed to notify the server about the stopped task %d: %v",
				agentAwareTask.TaskId, err)
			return
		}
	}()

	worker.logger.Infof("started task %d", agentAwareTask.TaskId)
}

func (worker *Worker) stopTask(taskID int64) {
	if cancel, ok := worker.tasks[taskID]; ok {
		cancel()
	}

	worker.logger.Infof("sent cancellation signal to task %d", taskID)
}

func (worker *Worker) runningTasks() (result []int64) {
	for taskID := range worker.tasks {
		result = append(result, taskID)
	}

	return
}

func (worker *Worker) registerTaskCompletions() {
	for {
		select {
		case taskID := <-worker.taskCompletions:
			if cancel, ok := worker.tasks[taskID]; ok {
				cancel()
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
