package build

import (
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/build/taskstatus"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"strconv"
	"sync"
	"time"
)

const defaultTaskTimeout = 60 * time.Minute

type Task struct {
	ID          int64
	RequiredIDs []int64
	Name        string
	status      taskstatus.Status
	Instance    instance.Instance
	Timeout     time.Duration
	Environment map[string]string
	Commands    []*Command

	// A mutex to guarantee safe accesses from both the main loop and gRPC server handlers
	Mutex sync.RWMutex
}

func NewFromProto(protoTask *api.Task) (*Task, error) {
	// Create an instance that this task will run on
	inst, err := instance.NewFromProto(protoTask.Instance, protoTask.Commands)
	if err != nil {
		return nil, err
	}

	// Intercept the first clone instruction and remove it
	for i, command := range protoTask.Commands {
		if command.Name == "clone" {
			protoTask.Commands = append(protoTask.Commands[:i], protoTask.Commands[i+1:]...)
			break
		}
	}

	var wrappedCommands []*Command
	for _, command := range protoTask.Commands {
		wrappedCommands = append(wrappedCommands, &Command{
			ProtoCommand: command,
		})
	}

	timeout := defaultTaskTimeout
	if protoTask.Metadata != nil {
		metadataTimeout, found := protoTask.Metadata.Properties["timeoutInSeconds"]
		if found {
			metadataTimeout, err := strconv.Atoi(metadataTimeout)
			if err != nil {
				return nil, err
			}
			timeout = time.Duration(metadataTimeout) * time.Second
		}
	}

	return &Task{
		ID:          protoTask.LocalGroupId,
		RequiredIDs: protoTask.RequiredGroups,
		Name:        protoTask.Name,
		Instance:    inst,
		Timeout:     timeout,
		Environment: protoTask.Environment,
		Commands:    wrappedCommands,
	}, nil
}

func (task *Task) ProtoCommands() []*api.Command {
	var result []*api.Command

	for _, command := range task.Commands {
		result = append(result, command.ProtoCommand)
	}

	return result
}

func (task *Task) FailedAtLeastOnce() bool {
	for _, command := range task.Commands {
		if command.Status() == taskstatus.Failed {
			return true
		}
	}

	return false
}

func (task *Task) Status() taskstatus.Status {
	task.Mutex.RLock()
	defer task.Mutex.RUnlock()

	// Task status is normally composed of it's command statuses, but if someone alters the default
	// value through Task.SetStatus() — we'll skip the calculation and return that value instead
	if task.status != taskstatus.New {
		return task.status
	}

	failedAtLeastOnce := task.FailedAtLeastOnce()

	for _, command := range task.Commands {
		shouldRun := (command.ProtoCommand.ExecutionBehaviour == api.Command_ON_SUCCESS && !failedAtLeastOnce) ||
			(command.ProtoCommand.ExecutionBehaviour == api.Command_ON_FAILURE && failedAtLeastOnce) ||
			(command.ProtoCommand.ExecutionBehaviour == api.Command_ALWAYS)

		if command.Status() == taskstatus.New && shouldRun {
			return taskstatus.New
		}
	}

	if failedAtLeastOnce {
		return taskstatus.Failed
	}

	return taskstatus.Succeeded
}

func (task *Task) SetStatus(status taskstatus.Status) {
	task.Mutex.Lock()
	defer task.Mutex.Unlock()

	task.status = status
}

func (task *Task) GetCommand(name string) *Command {
	for _, command := range task.Commands {
		if command.ProtoCommand.Name == name {
			return command
		}
	}

	return nil
}

func (task *Task) String() string {
	return fmt.Sprintf("%s (%d)", task.Name, task.ID)
}
