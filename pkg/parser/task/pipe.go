package task

import (
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"google.golang.org/protobuf/proto"
)

const (
	DefaultCPU    = 2.0
	DefaultMemory = 4096
)

type DockerPipe struct {
	proto api.Task

	alias     string
	dependsOn []string

	parseable.DefaultParser
}

func NewDockerPipe(env map[string]string) *DockerPipe {
	pipe := &DockerPipe{}

	pipe.CollectibleField("environment", schema.TodoSchema, func(node *node.Node) error {
		environment, err := node.GetStringMapping()
		if err != nil {
			return err
		}
		pipe.proto.Environment = environment
		return nil
	})
	pipe.CollectibleField("env", schema.TodoSchema, func(node *node.Node) error {
		environment, err := node.GetStringMapping()
		if err != nil {
			return err
		}
		pipe.proto.Environment = environment
		return nil
	})

	pipe.RequiredField(nameable.NewSimpleNameable("steps"), schema.TodoSchema, func(stepsNode *node.Node) error {
		if _, ok := stepsNode.Value.(*node.ListValue); !ok {
			return fmt.Errorf("%w: steps should be a list node", parsererror.ErrParsing)
		}

		for _, child := range stepsNode.Children {
			step := NewPipeStep()
			if err := step.Parse(child); err != nil {
				return err
			}
			pipe.proto.Commands = append(pipe.proto.Commands, step.protoCommands...)
		}

		return nil
	})

	return pipe
}

func (pipe *DockerPipe) Parse(node *node.Node) error {
	if err := pipe.DefaultParser.Parse(node); err != nil {
		return err
	}

	instance := &api.PipeInstance{
		Cpu:    DefaultCPU,
		Memory: DefaultMemory,
	}

	instanceBytes, err := proto.Marshal(instance)
	if err != nil {
		return err
	}

	pipe.proto.Instance = &api.Task_Instance{
		Type:    "pipe",
		Payload: instanceBytes,
	}

	return nil
}

func (pipe *DockerPipe) Name() string {
	if pipe.alias != "" {
		return pipe.alias
	}

	return pipe.proto.Name
}

func (pipe *DockerPipe) SetName(name string) {
	pipe.proto.Name = name
}

func (pipe *DockerPipe) DependsOnNames() []string {
	return pipe.dependsOn
}

func (pipe *DockerPipe) ID() int64      { return pipe.proto.LocalGroupId }
func (pipe *DockerPipe) SetID(id int64) { pipe.proto.LocalGroupId = id }

func (pipe *DockerPipe) DependsOnIDs() []int64       { return pipe.proto.RequiredGroups }
func (pipe *DockerPipe) SetDependsOnIDs(ids []int64) { pipe.proto.RequiredGroups = ids }

func (pipe *DockerPipe) Proto() interface{} {
	return &pipe.proto
}
