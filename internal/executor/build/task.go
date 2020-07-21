package build

import (
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"strconv"
	"time"
)

type TaskStatus int

const (
	StatusNew TaskStatus = iota
	StatusSucceeded
	StatusFailed
	StatusTimedOut
)

func (status TaskStatus) String() string {
	switch status {
	case StatusNew:
		return "new"
	case StatusSucceeded:
		return "succeeded"
	case StatusFailed:
		return "failed"
	case StatusTimedOut:
		return "timed out"
	default:
		return fmt.Sprintf("entered unhandled status %d", int(status))
	}
}

const defaultTaskTimeout = 60 * time.Minute

type Task struct {
	ID       int64
	Status   TaskStatus
	Instance *instance.Instance
	Timeout  time.Duration

	// Original Protocol Buffers structure for reference
	ProtoTask *api.Task
}

func NewFromProto(protoTask *api.Task) (*Task, error) {
	// Create an instance that this task will run on
	inst, err := instance.NewFromProto(protoTask.Instance)
	if err != nil {
		return nil, err
	}

	// Intercept the first clone instruction and adapt it for the CLI
	for _, command := range protoTask.Commands {
		_, ok := command.Instruction.(*api.Command_CloneInstruction)
		if !ok {
			continue
		}

		*command = api.Command{
			Name: command.Name,
			Instruction: &api.Command_ScriptInstruction{
				ScriptInstruction: &api.ScriptInstruction{
					Scripts: []string{"cp -rT $CIRRUS_PROJECT_DIR ."},
				},
			},
		}

		break
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

func (task *Task) String() string {
	return fmt.Sprintf("%s (%d)", task.ProtoTask.Name, task.ID)
}
