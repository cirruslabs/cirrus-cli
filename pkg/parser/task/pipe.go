package task

import (
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"github.com/golang/protobuf/ptypes"
	"strconv"
)

const (
	defaultCPU    = 2.0
	defaultMemory = 4096
)

type DockerPipe struct {
	proto api.Task

	alias     string
	dependsOn []string

	enabled bool

	parseable.DefaultParser
}

func NewDockerPipe(env map[string]string) *DockerPipe {
	pipe := &DockerPipe{
		enabled: true,
	}
	pipe.proto.Metadata = &api.Task_Metadata{Properties: map[string]string{}}

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
			step := NewPipeStep(environment.Merge(pipe.proto.Environment, env))
			if err := step.Parse(child); err != nil {
				return err
			}
			pipe.proto.Commands = append(pipe.proto.Commands, step.protoCommands...)
		}

		return nil
	})

	pipe.OptionalField(nameable.NewSimpleNameable("only_if"), schema.TodoSchema, func(node *node.Node) error {
		evaluation, err := handleBoolevatorField(node, environment.Merge(pipe.proto.Environment, env))
		if err != nil {
			return err
		}
		pipe.enabled = evaluation
		return nil
	})
	pipe.OptionalField(nameable.NewSimpleNameable("allow_failures"), schema.TodoSchema, func(node *node.Node) error {
		evaluation, err := handleBoolevatorField(node, environment.Merge(pipe.proto.Environment, env))
		if err != nil {
			return err
		}
		pipe.proto.Metadata.Properties["allowFailures"] = strconv.FormatBool(evaluation)
		return nil
	})

	return pipe
}

func (pipe *DockerPipe) Parse(node *node.Node) error {
	if err := pipe.DefaultParser.Parse(node); err != nil {
		return err
	}

	instance := &api.PipeInstance{
		Cpu:    defaultCPU,
		Memory: defaultMemory,
	}

	anyInstance, err := ptypes.MarshalAny(instance)
	if err != nil {
		return err
	}

	pipe.proto.Instance = anyInstance

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

func (pipe *DockerPipe) ID() int64 { return pipe.proto.LocalGroupId }
func (pipe *DockerPipe) SetID(id int64) {
	pipe.proto.LocalGroupId = id
	pipe.proto.Metadata.Properties["indexWithinBuild"] = strconv.FormatInt(id, 10)
}

func (pipe *DockerPipe) DependsOnIDs() []int64       { return pipe.proto.RequiredGroups }
func (pipe *DockerPipe) SetDependsOnIDs(ids []int64) { pipe.proto.RequiredGroups = ids }

func (pipe *DockerPipe) Proto() interface{} {
	return &pipe.proto
}

func (pipe *DockerPipe) Enabled() bool {
	return pipe.enabled
}
