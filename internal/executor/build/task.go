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
	ID       int64
	status   taskstatus.Status
	Instance instance.Instance
	Timeout  time.Duration

	// Original Protocol Buffers structure for reference
	ProtoTask *api.Task

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
		ID:        protoTask.LocalGroupId,
		Instance:  inst,
		Timeout:   timeout,
		ProtoTask: protoTask,
	}, nil
}

func (task *Task) Status() taskstatus.Status {
	task.Mutex.RLock()
	defer task.Mutex.RUnlock()

	return task.status
}

func (task *Task) SetStatus(status taskstatus.Status) {
	task.Mutex.Lock()
	defer task.Mutex.Unlock()

	task.status = status
}

func (task *Task) String() string {
	return fmt.Sprintf("%s (%d)", task.ProtoTask.Name, task.ID)
}
