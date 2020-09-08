package task

import (
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"google.golang.org/protobuf/proto"
	"strings"
)

type Task struct {
	proto api.Task

	alias     string
	dependsOn []string

	enabled bool

	parseable.DefaultParser
}

func NewTask(env map[string]string) *Task {
	task := &Task{
		enabled: true,
	}

	task.CollectibleField("environment", schema.TodoSchema, func(node *node.Node) error {
		taskEnv, err := node.GetStringMapping()
		if err != nil {
			return err
		}
		task.proto.Environment = taskEnv
		return nil
	})
	task.CollectibleField("env", schema.TodoSchema, func(node *node.Node) error {
		taskEnv, err := node.GetStringMapping()
		if err != nil {
			return err
		}
		task.proto.Environment = taskEnv
		return nil
	})

	task.CollectibleField("container",
		instance.NewCommunityContainer(environment.Merge(task.proto.Environment, env)).Schema(),
		func(node *node.Node) error {
			inst := instance.NewCommunityContainer(environment.Merge(task.proto.Environment, env))
			protoInstance, err := inst.Parse(node)
			if err != nil {
				return err
			}
			task.proto.Instance = protoInstance

			// Retrieve the platform to update the base environment
			var taskContainer api.ContainerInstance
			if err := proto.Unmarshal(protoInstance.Payload, &taskContainer); err != nil {
				return err
			}
			env["CIRRUS_OS"] = strings.ToLower(taskContainer.Platform.String())

			return nil
		})

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

	bgNameable := nameable.NewRegexNameable("(.*)background_script")
	task.OptionalField(bgNameable, schema.TodoSchema, func(node *node.Node) error {
		scripts, err := node.GetSliceOfNonEmptyStrings()
		if err != nil {
			return err
		}
		task.proto.Commands = append(task.proto.Commands, &api.Command{
			Name: bgNameable.FirstGroupOrDefault(node.Name, "main"),
			Instruction: &api.Command_BackgroundScriptInstruction{
				BackgroundScriptInstruction: &api.BackgroundScriptInstruction{
					Scripts: scripts,
				},
			},
		})
		return nil
	})

	scriptNameable := nameable.NewRegexNameable("(.*)script")
	task.OptionalField(scriptNameable, schema.TodoSchema, func(node *node.Node) error {
		scripts, err := node.GetSliceOfNonEmptyStrings()
		if err != nil {
			return err
		}
		task.proto.Commands = append(task.proto.Commands, &api.Command{
			Name: scriptNameable.FirstGroupOrDefault(node.Name, "main"),
			Instruction: &api.Command_ScriptInstruction{
				ScriptInstruction: &api.ScriptInstruction{
					Scripts: scripts,
				},
			},
		})
		return nil
	})

	cacheNameable := nameable.NewRegexNameable("(.*)cache")
	task.OptionalField(cacheNameable, schema.TodoSchema, func(node *node.Node) error {
		cache := NewCacheCommand(environment.Merge(task.proto.Environment, env))
		if err := cache.Parse(node); err != nil {
			return err
		}
		task.proto.Commands = append(task.proto.Commands, cache.Proto())
		return nil
	})

	for id, name := range api.Command_CommandExecutionBehavior_name {
		idCopy := id
		task.OptionalField(nameable.NewSimpleNameable(strings.ToLower(name)), schema.TodoSchema, func(node *node.Node) error {
			behavior := NewBehavior()
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

	task.OptionalField(nameable.NewSimpleNameable("depends_on"), schema.TodoSchema, func(node *node.Node) error {
		dependsOn, err := node.GetSliceOfNonEmptyStrings()
		if err != nil {
			return err
		}
		task.dependsOn = dependsOn
		return nil
	})

	task.OptionalField(nameable.NewSimpleNameable("only_if"), schema.TodoSchema, func(node *node.Node) error {
		evaluation, err := handleOnlyIf(node, environment.Merge(task.proto.Environment, env))
		if err != nil {
			return err
		}
		task.enabled = evaluation
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

func (task *Task) ID() int64      { return task.proto.LocalGroupId }
func (task *Task) SetID(id int64) { task.proto.LocalGroupId = id }

func (task *Task) DependsOnIDs() []int64       { return task.proto.RequiredGroups }
func (task *Task) SetDependsOnIDs(ids []int64) { task.proto.RequiredGroups = ids }

func (task *Task) Proto() interface{} {
	return &task.proto
}

func (task *Task) Enabled() bool {
	return task.enabled
}
