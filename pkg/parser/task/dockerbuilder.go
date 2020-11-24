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
	"github.com/cirruslabs/cirrus-cli/pkg/parser/task/command"
	"github.com/golang/protobuf/ptypes"
	jsschema "github.com/lestrrat-go/jsschema"
	"strconv"
	"strings"
)

type DockerBuilder struct {
	proto api.Task

	platform  api.Platform
	osVersion string

	alias     string
	dependsOn []string

	onlyIfExpression string

	parseable.DefaultParser
}

// nolint:gocognit // it's a parser, there is a lot of boilerplate
func NewDockerBuilder(env map[string]string, boolevator *boolevator.Boolevator) *DockerBuilder {
	dbuilder := &DockerBuilder{}
	dbuilder.proto.Environment = map[string]string{"CIRRUS_OS": "linux"}
	dbuilder.proto.Metadata = &api.Task_Metadata{Properties: DefaultTaskProperties()}

	dbuilder.CollectibleField("environment", schema.Map(""), func(node *node.Node) error {
		taskEnv, err := node.GetEnvironment()
		if err != nil {
			return err
		}
		dbuilder.proto.Environment = environment.Merge(dbuilder.proto.Environment, taskEnv)
		return nil
	})

	dbuilder.CollectibleField("env", schema.Map(""), func(node *node.Node) error {
		taskEnv, err := node.GetEnvironment()
		if err != nil {
			return err
		}
		dbuilder.proto.Environment = environment.Merge(dbuilder.proto.Environment, taskEnv)
		return nil
	})

	dbuilder.OptionalField(nameable.NewSimpleNameable("name"), schema.String(""), func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(environment.Merge(dbuilder.proto.Environment, env))
		if err != nil {
			return err
		}
		dbuilder.proto.Name = name
		return nil
	})

	dbuilder.OptionalField(nameable.NewSimpleNameable("alias"), schema.String(""), func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(environment.Merge(dbuilder.proto.Environment, env))
		if err != nil {
			return err
		}
		dbuilder.alias = name
		return nil
	})

	bgNameable := nameable.NewRegexNameable("^(.*)background_script$")
	dbuilder.OptionalField(bgNameable, schema.Script(""), func(node *node.Node) error {
		command, err := handleBackgroundScript(node, bgNameable)
		if err != nil {
			return err
		}

		dbuilder.proto.Commands = append(dbuilder.proto.Commands, command)

		return nil
	})

	scriptNameable := nameable.NewRegexNameable("^(.*)script$")
	dbuilder.OptionalField(scriptNameable, schema.Script(""), func(node *node.Node) error {
		command, err := handleScript(node, scriptNameable)
		if err != nil {
			return err
		}

		dbuilder.proto.Commands = append(dbuilder.proto.Commands, command)

		return nil
	})

	cacheNameable := nameable.NewRegexNameable("^(.*)cache$")
	cacheSchema := command.NewCacheCommand(nil, nil).Schema()
	dbuilder.OptionalField(cacheNameable, cacheSchema, func(node *node.Node) error {
		cache := command.NewCacheCommand(environment.Merge(dbuilder.proto.Environment, env), boolevator)
		if err := cache.Parse(node); err != nil {
			return err
		}
		dbuilder.proto.Commands = append(dbuilder.proto.Commands, cache.Proto())
		return nil
	})

	artifactsNameable := nameable.NewRegexNameable("^(.*)artifacts$")
	artifactsSchema := command.NewArtifactsCommand(nil).Schema()
	dbuilder.OptionalField(artifactsNameable, artifactsSchema, func(node *node.Node) error {
		artifacts := command.NewArtifactsCommand(environment.Merge(dbuilder.proto.Environment, env))
		if err := artifacts.Parse(node); err != nil {
			return err
		}
		dbuilder.proto.Commands = append(dbuilder.proto.Commands, artifacts.Proto())
		return nil
	})

	fileNameable := nameable.NewRegexNameable("^(.*)file$")
	fileSchema := command.NewFileCommand(nil).Schema()
	dbuilder.OptionalField(fileNameable, fileSchema, func(node *node.Node) error {
		file := command.NewFileCommand(environment.Merge(dbuilder.proto.Environment, env))
		if err := file.Parse(node); err != nil {
			return err
		}
		dbuilder.proto.Commands = append(dbuilder.proto.Commands, file.Proto())
		return nil
	})

	dbuilder.registerExecutionBehaviorFields(env, boolevator)

	dependsOnSchema := schema.StringOrListOfStrings("List of task names this task depends on.")
	dbuilder.OptionalField(nameable.NewSimpleNameable("depends_on"), dependsOnSchema, func(node *node.Node) error {
		dependsOn, err := node.GetSliceOfNonEmptyStrings()
		if err != nil {
			return err
		}
		dbuilder.dependsOn = dependsOn
		return nil
	})

	dbuilder.CollectibleField("only_if", schema.Condition(""), func(node *node.Node) error {
		onlyIfExpression, err := node.GetStringValue()
		if err != nil {
			return err
		}
		dbuilder.onlyIfExpression = onlyIfExpression
		return nil
	})

	dbuilder.CollectibleField("skip", schema.Condition(""), func(node *node.Node) error {
		skipped, err := node.GetBoolValue(environment.Merge(dbuilder.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		if skipped {
			dbuilder.proto.Status = api.Task_SKIPPED
		}
		return nil
	})

	dbuilder.CollectibleField("allow_failures", schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(dbuilder.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		dbuilder.proto.Metadata.Properties["allow_failures"] = strconv.FormatBool(evaluation)
		return nil
	})

	// for cloud only
	dbuilder.CollectibleField("skip_notifications", schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(dbuilder.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		dbuilder.proto.Metadata.Properties["skip_notifications"] = strconv.FormatBool(evaluation)
		return nil
	})

	// for cloud only
	dbuilder.CollectibleField("auto_cancellation", schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(dbuilder.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		dbuilder.proto.Metadata.Properties["auto_cancellation"] = strconv.FormatBool(evaluation)
		return nil
	})

	// for cloud only
	dbuilder.CollectibleField("use_compute_credits", schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(dbuilder.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		dbuilder.proto.Metadata.Properties["use_compute_credits"] = strconv.FormatBool(evaluation)
		return nil
	})

	// for cloud only
	dbuilder.CollectibleField("stateful", schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(dbuilder.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		dbuilder.proto.Metadata.Properties["stateful"] = strconv.FormatBool(evaluation)
		return nil
	})

	// no-op
	labelsSchema := schema.StringOrListOfStrings("List of required labels on a PR.")
	dbuilder.OptionalField(nameable.NewSimpleNameable("required_pr_labels"), labelsSchema, func(node *node.Node) error {
		return nil
	})

	dbuilder.CollectibleField("experimental", schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(dbuilder.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		dbuilder.proto.Metadata.Properties["experimental"] = strconv.FormatBool(evaluation)
		return nil
	})

	dbuilder.CollectibleField("timeout_in", schema.Number("Task timeout in minutes"), func(node *node.Node) error {
		timeout, err := handleTimeoutIn(node, environment.Merge(dbuilder.proto.Environment, env))
		if err != nil {
			return err
		}

		dbuilder.proto.Metadata.Properties["timeout_in"] = timeout

		return nil
	})

	dbuilder.CollectibleField("trigger_type", schema.TriggerType(), func(node *node.Node) error {
		triggerType, err := node.GetExpandedStringValue(environment.Merge(dbuilder.proto.Environment, env))
		if err != nil {
			return err
		}
		dbuilder.proto.Metadata.Properties["trigger_type"] = strings.ToUpper(triggerType)
		return nil
	})

	lockSchema := schema.String("Lock name for triggering and execution")
	dbuilder.OptionalField(nameable.NewSimpleNameable("execution_lock"), lockSchema, func(node *node.Node) error {
		lockName, err := node.GetExpandedStringValue(environment.Merge(dbuilder.proto.Environment, env))
		if err != nil {
			return err
		}

		dbuilder.proto.Metadata.Properties["execution_lock"] = lockName

		return nil
	})

	// no-op
	sipSchema := schema.Condition("")
	dbuilder.OptionalField(nameable.NewSimpleNameable("use_static_ip"), sipSchema, func(node *node.Node) error {
		return nil
	})

	platformSchema := schema.Enum([]interface{}{"linux", "windows"}, "Container Platform.")
	dbuilder.OptionalField(nameable.NewSimpleNameable("platform"), platformSchema, func(node *node.Node) error {
		platformName, err := node.GetExpandedStringValue(environment.Merge(dbuilder.proto.Environment, env))
		if err != nil {
			return err
		}

		platformValue, ok := api.Platform_value[strings.ToUpper(platformName)]
		if !ok {
			return fmt.Errorf("%w: unknown platform name %q", parsererror.ErrParsing, platformName)
		}

		dbuilder.platform = api.Platform(platformValue)
		dbuilder.proto.Environment["CIRRUS_OS"] = platformName

		return nil
	})

	osVersionSchema := schema.Enum([]interface{}{"2019", "1709", "1803"}, "Windows version of container.")
	dbuilder.OptionalField(nameable.NewSimpleNameable("os_version"), osVersionSchema, func(node *node.Node) error {
		osVersion, err := node.GetExpandedStringValue(environment.Merge(dbuilder.proto.Environment, env))
		if err != nil {
			return err
		}

		dbuilder.osVersion = osVersion

		return nil
	})

	return dbuilder
}

func (dbuilder *DockerBuilder) registerExecutionBehaviorFields(
	env map[string]string,
	boolevator *boolevator.Boolevator,
) {
	for id, name := range api.Command_CommandExecutionBehavior_name {
		idCopy := id

		behaviorNameable := nameable.NewSimpleNameable(strings.ToLower(name))
		behaviorSchema := NewBehavior(nil, nil).Schema()
		behaviorSchema.Description = name + " commands."
		dbuilder.OptionalField(behaviorNameable, behaviorSchema, func(node *node.Node) error {
			behavior := NewBehavior(environment.Merge(dbuilder.proto.Environment, env), boolevator)
			if err := behavior.Parse(node); err != nil {
				return err
			}

			commands := behavior.Proto()

			for _, command := range commands {
				command.ExecutionBehaviour = api.Command_CommandExecutionBehavior(idCopy)
				dbuilder.proto.Commands = append(dbuilder.proto.Commands, command)
			}

			return nil
		})
	}
}

func (dbuilder *DockerBuilder) Parse(node *node.Node) error {
	if err := dbuilder.DefaultParser.Parse(node); err != nil {
		return err
	}

	instance := &api.DockerBuilder{
		Platform:  dbuilder.platform,
		OsVersion: dbuilder.osVersion,
	}

	anyInstance, err := ptypes.MarshalAny(instance)
	if err != nil {
		return err
	}

	dbuilder.proto.Instance = anyInstance

	return nil
}

func (dbuilder *DockerBuilder) Name() string {
	if dbuilder.alias != "" {
		return dbuilder.alias
	}

	return dbuilder.proto.Name
}

func (dbuilder *DockerBuilder) SetName(name string) {
	dbuilder.proto.Name = name
}

func (dbuilder *DockerBuilder) DependsOnNames() []string {
	return dbuilder.dependsOn
}

func (dbuilder *DockerBuilder) ID() int64 { return dbuilder.proto.LocalGroupId }
func (dbuilder *DockerBuilder) SetID(id int64) {
	dbuilder.proto.LocalGroupId = id
}

func (dbuilder *DockerBuilder) SetIndexWithinBuild(id int64) {
	dbuilder.proto.Metadata.Properties["indexWithinBuild"] = strconv.FormatInt(id, 10)
}

func (dbuilder *DockerBuilder) Proto() interface{} {
	return &dbuilder.proto
}

func (dbuilder *DockerBuilder) DependsOnIDs() []int64       { return dbuilder.proto.RequiredGroups }
func (dbuilder *DockerBuilder) SetDependsOnIDs(ids []int64) { dbuilder.proto.RequiredGroups = ids }

func (dbuilder *DockerBuilder) Enabled(env map[string]string, boolevator *boolevator.Boolevator) (bool, error) {
	if dbuilder.onlyIfExpression == "" {
		return true, nil
	}

	evaluation, err := boolevator.Eval(dbuilder.onlyIfExpression, environment.Merge(dbuilder.proto.Environment, env))
	if err != nil {
		return false, err
	}

	return evaluation, nil
}

func (dbuilder *DockerBuilder) Schema() *jsschema.Schema {
	modifiedSchema := dbuilder.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}

	return modifiedSchema
}
