package task

import (
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/task/command"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"strconv"
	"strings"
)

type Task struct {
	proto api.Task

	alias     string
	dependsOn []string

	onlyIfExpression string

	parseable.DefaultParser
}

//nolint:gocognit // it's a parser, there is a lot of boilerplate
func NewTask(env map[string]string, additionalInstances map[string]protoreflect.MessageDescriptor) *Task {
	task := &Task{}
	task.proto.Metadata = &api.Task_Metadata{Properties: DefaultTaskProperties()}

	task.CollectibleField("environment", schema.TodoSchema, func(node *node.Node) error {
		taskEnv, err := node.GetStringMapping()
		if err != nil {
			return err
		}
		task.proto.Environment = environment.Merge(task.proto.Environment, taskEnv)
		return nil
	})
	task.CollectibleField("env", schema.TodoSchema, func(node *node.Node) error {
		taskEnv, err := node.GetStringMapping()
		if err != nil {
			return err
		}
		task.proto.Environment = environment.Merge(task.proto.Environment, taskEnv)
		return nil
	})

	task.CollectibleField("container",
		instance.NewCommunityContainer(environment.Merge(task.proto.Environment, env)).Schema(),
		func(node *node.Node) error {
			inst := instance.NewCommunityContainer(environment.Merge(task.proto.Environment, env))
			containerInstance, err := inst.Parse(node)
			if err != nil {
				return err
			}

			// Retrieve the platform to update the base environment
			env["CIRRUS_OS"] = strings.ToLower(containerInstance.Platform.String())

			anyInstance, err := ptypes.MarshalAny(containerInstance)
			if err != nil {
				return err
			}
			task.proto.Instance = anyInstance

			return nil
		})

	for instanceName, descriptor := range additionalInstances {
		additionalInstanceParser := instance.NewProtoParser(descriptor, environment.Merge(task.proto.Environment, env))
		task.CollectibleField(instanceName,
			additionalInstanceParser.Schema(),
			func(node *node.Node) error {
				parserInstance, err := additionalInstanceParser.Parse(node)
				if err != nil {
					return err
				}
				anyInstance, err := anypb.New(parserInstance)
				if err != nil {
					return err
				}
				task.proto.Instance = anyInstance
				return nil
			})
	}

	task.OptionalField(nameable.NewSimpleNameable("name"), schema.TodoSchema, func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(environment.Merge(task.proto.Environment, env))
		if err != nil {
			return err
		}
		task.proto.Name = name
		return nil
	})
	task.OptionalField(nameable.NewSimpleNameable("alias"), schema.TodoSchema, func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(environment.Merge(task.proto.Environment, env))
		if err != nil {
			return err
		}
		task.alias = name
		return nil
	})

	bgNameable := nameable.NewRegexNameable("^(.*)background_script$")
	task.OptionalField(bgNameable, schema.TodoSchema, func(node *node.Node) error {
		command, err := handleBackgroundScript(node, bgNameable)
		if err != nil {
			return err
		}

		task.proto.Commands = append(task.proto.Commands, command)

		return nil
	})

	scriptNameable := nameable.NewRegexNameable("^(.*)script$")
	task.OptionalField(scriptNameable, schema.TodoSchema, func(node *node.Node) error {
		command, err := handleScript(node, scriptNameable)
		if err != nil {
			return err
		}

		task.proto.Commands = append(task.proto.Commands, command)

		return nil
	})

	cacheNameable := nameable.NewRegexNameable("^(.*)cache$")
	task.OptionalField(cacheNameable, schema.TodoSchema, func(node *node.Node) error {
		cache := NewCacheCommand(environment.Merge(task.proto.Environment, env))
		if err := cache.Parse(node); err != nil {
			return err
		}
		task.proto.Commands = append(task.proto.Commands, cache.Proto())
		return nil
	})

	artifactsNameable := nameable.NewRegexNameable("^(.*)artifacts$")
	task.OptionalField(artifactsNameable, schema.TodoSchema, func(node *node.Node) error {
		artifacts := command.NewArtifactsCommand(environment.Merge(task.proto.Environment, env))
		if err := artifacts.Parse(node); err != nil {
			return err
		}
		task.proto.Commands = append(task.proto.Commands, artifacts.Proto())
		return nil
	})

	task.registerExecutionBehaviorFields(env)

	task.OptionalField(nameable.NewSimpleNameable("depends_on"), schema.TodoSchema, func(node *node.Node) error {
		dependsOn, err := node.GetSliceOfNonEmptyStrings()
		if err != nil {
			return err
		}
		task.dependsOn = dependsOn
		return nil
	})

	task.CollectibleField("only_if", schema.TodoSchema, func(node *node.Node) error {
		onlyIfExpression, err := node.GetStringValue()
		if err != nil {
			return err
		}
		task.onlyIfExpression = onlyIfExpression
		return nil
	})

	task.CollectibleField("allow_failures", schema.TodoSchema, func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(task.proto.Environment, env))
		if err != nil {
			return err
		}
		task.proto.Metadata.Properties["allowFailures"] = strconv.FormatBool(evaluation)
		return nil
	})

	task.CollectibleField("experimental", schema.TodoSchema, func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(task.proto.Environment, env))
		if err != nil {
			return err
		}
		task.proto.Metadata.Properties["experimentalFeaturesEnabled"] = strconv.FormatBool(evaluation)
		return nil
	})

	task.CollectibleField("timeout_in", schema.TodoSchema, func(node *node.Node) error {
		timeoutInSeconds, err := handleTimeoutIn(node, environment.Merge(task.proto.Environment, env))
		if err != nil {
			return err
		}

		task.proto.Metadata.Properties["timeoutInSeconds"] = timeoutInSeconds

		return nil
	})

	task.CollectibleField("trigger_type", schema.TodoSchema, func(node *node.Node) error {
		triggerType, err := node.GetExpandedStringValue(environment.Merge(task.proto.Environment, env))
		if err != nil {
			return err
		}
		task.proto.Metadata.Properties["triggerType"] = strings.ToUpper(triggerType)
		return nil
	})

	task.OptionalField(nameable.NewSimpleNameable("execution_lock"), schema.TodoSchema, func(node *node.Node) error {
		lockName, err := node.GetExpandedStringValue(environment.Merge(task.proto.Environment, env))
		if err != nil {
			return err
		}

		task.proto.Metadata.Properties["executionLock"] = lockName

		return nil
	})

	return task
}

func (task *Task) Parse(node *node.Node) error {
	if err := task.DefaultParser.Parse(node); err != nil {
		return err
	}

	if task.proto.Instance == nil {
		return fmt.Errorf("%w: task has no instance attached", parsererror.ErrParsing)
	}

	// Generate cache upload instructions
	for _, command := range task.proto.Commands {
		_, ok := command.Instruction.(*api.Command_CacheInstruction)
		if !ok {
			continue
		}

		uploadCommand := &api.Command{
			Name: fmt.Sprintf("Upload '%s' cache", command.Name),
			Instruction: &api.Command_UploadCacheInstruction{
				UploadCacheInstruction: &api.UploadCacheInstruction{
					CacheName: command.Name,
				},
			},
			ExecutionBehaviour: command.ExecutionBehaviour,
		}

		task.proto.Commands = append(task.proto.Commands, uploadCommand)
	}

	return nil
}

func (task *Task) Name() string {
	if task.alias != "" {
		return task.alias
	}

	return task.proto.Name
}

func (task *Task) SetName(name string) {
	task.proto.Name = name
}

func (task *Task) DependsOnNames() []string {
	return task.dependsOn
}

func (task *Task) ID() int64 { return task.proto.LocalGroupId }
func (task *Task) SetID(id int64) {
	task.proto.LocalGroupId = id
	task.proto.Metadata.Properties["indexWithinBuild"] = strconv.FormatInt(id, 10)
}

func (task *Task) DependsOnIDs() []int64       { return task.proto.RequiredGroups }
func (task *Task) SetDependsOnIDs(ids []int64) { task.proto.RequiredGroups = ids }

func (task *Task) Proto() interface{} {
	return &task.proto
}

func (task *Task) Enabled(env map[string]string) (bool, error) {
	if task.onlyIfExpression == "" {
		return true, nil
	}

	evaluation, err := boolevator.Eval(task.onlyIfExpression, environment.Merge(task.proto.Environment, env), nil)
	if err != nil {
		return false, err
	}

	return evaluation, nil
}

func (task *Task) registerExecutionBehaviorFields(env map[string]string) {
	for id, name := range api.Command_CommandExecutionBehavior_name {
		idCopy := id
		task.OptionalField(nameable.NewSimpleNameable(strings.ToLower(name)), schema.TodoSchema, func(node *node.Node) error {
			behavior := NewBehavior(environment.Merge(task.proto.Environment, env))
			if err := behavior.Parse(node); err != nil {
				return err
			}

			commands := behavior.Proto()

			for _, command := range commands {
				command.ExecutionBehaviour = api.Command_CommandExecutionBehavior(idCopy)
				task.proto.Commands = append(task.proto.Commands, command)
			}

			return nil
		})
	}
}
