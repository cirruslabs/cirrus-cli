package task

import (
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"github.com/golang/protobuf/ptypes"
	"strconv"
	"strings"
)

const (
	defaultCPU    = 2.0
	defaultMemory = 4096
)

type DockerPipe struct {
	proto api.Task

	res *PipeResources

	alias     string
	dependsOn []string

	onlyIfExpression string

	parseable.DefaultParser
}

func NewDockerPipe(env map[string]string, boolevator *boolevator.Boolevator) *DockerPipe {
	pipe := &DockerPipe{}
	pipe.proto.Metadata = &api.Task_Metadata{Properties: DefaultTaskProperties()}

	pipe.CollectibleField("environment", schema.TodoSchema, func(node *node.Node) error {
		pipeEnv, err := node.GetEnvironment()
		if err != nil {
			return err
		}
		pipe.proto.Environment = environment.Merge(pipe.proto.Environment, pipeEnv)
		return nil
	})
	pipe.CollectibleField("env", schema.TodoSchema, func(node *node.Node) error {
		pipeEnv, err := node.GetEnvironment()
		if err != nil {
			return err
		}
		pipe.proto.Environment = environment.Merge(pipe.proto.Environment, pipeEnv)
		return nil
	})

	pipe.OptionalField(nameable.NewSimpleNameable("name"), schema.TodoSchema, func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(environment.Merge(pipe.proto.Environment, env))
		if err != nil {
			return err
		}
		pipe.proto.Name = name
		return nil
	})
	pipe.OptionalField(nameable.NewSimpleNameable("alias"), schema.TodoSchema, func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(environment.Merge(pipe.proto.Environment, env))
		if err != nil {
			return err
		}
		pipe.alias = name
		return nil
	})

	pipe.OptionalField(nameable.NewSimpleNameable("resources"), schema.TodoSchema, func(node *node.Node) error {
		resources := NewPipeResources(environment.Merge(pipe.proto.Environment, env))

		if err := resources.Parse(node); err != nil {
			return err
		}

		pipe.res = resources

		return nil
	})

	pipe.RequiredField(nameable.NewSimpleNameable("steps"), schema.TodoSchema, func(stepsNode *node.Node) error {
		if _, ok := stepsNode.Value.(*node.ListValue); !ok {
			return fmt.Errorf("%w: steps should be a list node", parsererror.ErrParsing)
		}

		for _, child := range stepsNode.Children {
			step := NewPipeStep(environment.Merge(pipe.proto.Environment, env), boolevator)
			if err := step.Parse(child); err != nil {
				return err
			}
			pipe.proto.Commands = append(pipe.proto.Commands, step.protoCommands...)
		}

		return nil
	})

	pipe.CollectibleField("only_if", schema.TodoSchema, func(node *node.Node) error {
		onlyIfExpression, err := node.GetStringValue()
		if err != nil {
			return err
		}
		pipe.onlyIfExpression = onlyIfExpression
		return nil
	})
	pipe.OptionalField(nameable.NewSimpleNameable("allow_failures"), schema.TodoSchema, func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(pipe.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		pipe.proto.Metadata.Properties["allow_failures"] = strconv.FormatBool(evaluation)
		return nil
	})

	pipe.OptionalField(nameable.NewSimpleNameable("experimental"), schema.TodoSchema, func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(pipe.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		pipe.proto.Metadata.Properties["experimental"] = strconv.FormatBool(evaluation)
		return nil
	})

	pipe.CollectibleField("timeout_in", schema.TodoSchema, func(node *node.Node) error {
		timeout_in, err := handleTimeoutIn(node, environment.Merge(pipe.proto.Environment, env))
		if err != nil {
			return err
		}

		pipe.proto.Metadata.Properties["timeout_in"] = timeout_in

		return nil
	})

	pipe.CollectibleField("trigger_type", schema.TodoSchema, func(node *node.Node) error {
		trigger_type, err := node.GetExpandedStringValue(environment.Merge(pipe.proto.Environment, env))
		if err != nil {
			return err
		}
		pipe.proto.Metadata.Properties["trigger_type"] = strings.ToUpper(trigger_type)
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

	// Pick up user-specified resource limits (if any)
	if pipe.res != nil {
		if pipe.res.cpu != 0 {
			instance.Cpu = pipe.res.cpu
		}
		if pipe.res.memory != 0 {
			instance.Memory = pipe.res.memory
		}
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

func (pipe *DockerPipe) Enabled(env map[string]string, boolevator *boolevator.Boolevator) (bool, error) {
	if pipe.onlyIfExpression == "" {
		return true, nil
	}

	evaluation, err := boolevator.Eval(pipe.onlyIfExpression, environment.Merge(pipe.proto.Environment, env))
	if err != nil {
		return false, err
	}

	return evaluation, nil
}
