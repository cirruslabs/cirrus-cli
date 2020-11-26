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
	jsschema "github.com/lestrrat-go/jsschema"
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

//nolint:gocognit // it's a parser, there is a lot of boilerplate
func NewDockerPipe(env map[string]string, boolevator *boolevator.Boolevator) *DockerPipe {
	pipe := &DockerPipe{}
	pipe.proto.Environment = map[string]string{"CIRRUS_OS": "linux"}
	pipe.proto.Metadata = &api.Task_Metadata{Properties: DefaultTaskProperties()}

	// Don't force required fields in schema
	pipe.SetCollectible(true)

	autoCancellation := env["CIRRUS_BRANCH"] != env["CIRRUS_DEFAULT_BRANCH"]
	if autoCancellation {
		pipe.proto.Metadata.Properties["auto_cancellation"] = strconv.FormatBool(autoCancellation)
	}

	pipe.CollectibleField("environment", schema.Map(""), func(node *node.Node) error {
		pipeEnv, err := node.GetEnvironment()
		if err != nil {
			return err
		}
		pipe.proto.Environment = environment.Merge(pipe.proto.Environment, pipeEnv)
		return nil
	})

	pipe.CollectibleField("env", schema.Map(""), func(node *node.Node) error {
		pipeEnv, err := node.GetEnvironment()
		if err != nil {
			return err
		}
		pipe.proto.Environment = environment.Merge(pipe.proto.Environment, pipeEnv)
		return nil
	})

	pipe.OptionalField(nameable.NewSimpleNameable("name"), schema.String(""), func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(environment.Merge(pipe.proto.Environment, env))
		if err != nil {
			return err
		}
		pipe.proto.Name = name
		return nil
	})
	pipe.OptionalField(nameable.NewSimpleNameable("alias"), schema.String(""), func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(environment.Merge(pipe.proto.Environment, env))
		if err != nil {
			return err
		}
		pipe.alias = name
		return nil
	})

	resourcesSchema := NewPipeResources(nil).Schema()
	pipe.OptionalField(nameable.NewSimpleNameable("resources"), resourcesSchema, func(node *node.Node) error {
		resources := NewPipeResources(environment.Merge(pipe.proto.Environment, env))

		if err := resources.Parse(node); err != nil {
			return err
		}

		pipe.res = resources

		return nil
	})

	stepsSchema := schema.ArrayOf(NewPipeStep(nil, nil).Schema())
	pipe.RequiredField(nameable.NewSimpleNameable("steps"), stepsSchema, func(stepsNode *node.Node) error {
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

	dependsSchema := schema.StringOrListOfStrings("List of task names this task depends on.")
	pipe.OptionalField(nameable.NewSimpleNameable("depends_on"), dependsSchema, func(node *node.Node) error {
		dependsOn, err := node.GetSliceOfNonEmptyStrings()
		if err != nil {
			return err
		}
		pipe.dependsOn = dependsOn
		return nil
	})

	pipe.CollectibleField("only_if", schema.Condition(""), func(node *node.Node) error {
		onlyIfExpression, err := node.GetStringValue()
		if err != nil {
			return err
		}
		pipe.onlyIfExpression = onlyIfExpression
		return nil
	})

	allowFailuresSchema := schema.Condition("")
	pipe.OptionalField(nameable.NewSimpleNameable("allow_failures"), allowFailuresSchema, func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(pipe.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		pipe.proto.Metadata.Properties["allow_failures"] = strconv.FormatBool(evaluation)
		return nil
	})

	// for cloud only
	pipe.CollectibleField("skip_notifications", schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(pipe.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		pipe.proto.Metadata.Properties["skip_notifications"] = strconv.FormatBool(evaluation)
		return nil
	})

	// for cloud only
	pipe.CollectibleField("auto_cancellation", schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(pipe.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		pipe.proto.Metadata.Properties["auto_cancellation"] = strconv.FormatBool(evaluation)
		return nil
	})

	// for cloud only
	pipe.CollectibleField("use_compute_credits", schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(pipe.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		pipe.proto.Metadata.Properties["use_compute_credits"] = strconv.FormatBool(evaluation)
		return nil
	})

	// for cloud only
	pipe.CollectibleField("stateful", schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(pipe.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		pipe.proto.Metadata.Properties["stateful"] = strconv.FormatBool(evaluation)
		return nil
	})

	// no-op
	rplSchema := schema.StringOrListOfStrings("List of required labels on a PR.")
	pipe.OptionalField(nameable.NewSimpleNameable("required_pr_labels"), rplSchema, func(node *node.Node) error {
		return nil
	})

	pipe.OptionalField(nameable.NewSimpleNameable("experimental"), schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(pipe.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		pipe.proto.Metadata.Properties["experimental"] = strconv.FormatBool(evaluation)
		return nil
	})

	pipe.CollectibleField("timeout_in", schema.Number("Task timeout in minutes"), func(node *node.Node) error {
		timeoutIn, err := handleTimeoutIn(node, environment.Merge(pipe.proto.Environment, env))
		if err != nil {
			return err
		}

		pipe.proto.Metadata.Properties["timeout_in"] = timeoutIn

		return nil
	})

	pipe.CollectibleField("trigger_type", schema.TriggerType(), func(node *node.Node) error {
		triggerType, err := node.GetExpandedStringValue(environment.Merge(pipe.proto.Environment, env))
		if err != nil {
			return err
		}
		pipe.proto.Metadata.Properties["trigger_type"] = strings.ToUpper(triggerType)
		return nil
	})

	lockSchema := schema.String("Lock name for triggering and execution")
	pipe.OptionalField(nameable.NewSimpleNameable("execution_lock"), lockSchema, func(node *node.Node) error {
		lockName, err := node.GetExpandedStringValue(environment.Merge(pipe.proto.Environment, env))
		if err != nil {
			return err
		}

		pipe.proto.Metadata.Properties["execution_lock"] = lockName

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
}

func (pipe *DockerPipe) SetIndexWithinBuild(id int64) {
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

func (pipe *DockerPipe) Schema() *jsschema.Schema {
	modifiedSchema := pipe.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}

	return modifiedSchema
}
