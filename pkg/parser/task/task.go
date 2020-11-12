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
	jsschema "github.com/lestrrat-go/jsschema"
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

// nolint:gocognit,gocyclo // it's a parser, there is a lot of boilerplate
func NewTask(
	env map[string]string,
	boolevator *boolevator.Boolevator,
	additionalInstances map[string]protoreflect.MessageDescriptor,
) *Task {
	task := &Task{}
	task.proto.Metadata = &api.Task_Metadata{Properties: DefaultTaskProperties()}

	// Don't force required fields in schema
	task.SetCollectible(true)

	task.proto.Metadata.Properties["auto_cancellation"] =
		strconv.FormatBool(env["CIRRUS_BRANCH"] != env["CIRRUS_DEFAULT_BRANCH"])

	task.CollectibleField("environment", schema.Map(""), func(node *node.Node) error {
		taskEnv, err := node.GetEnvironment()
		if err != nil {
			return err
		}
		task.proto.Environment = environment.Merge(task.proto.Environment, taskEnv)
		return nil
	})

	task.CollectibleField("env", schema.Map(""), func(node *node.Node) error {
		taskEnv, err := node.GetEnvironment()
		if err != nil {
			return err
		}
		task.proto.Environment = environment.Merge(task.proto.Environment, taskEnv)
		return nil
	})

	if _, ok := additionalInstances["container"]; !ok {
		task.CollectibleField("container",
			instance.NewCommunityContainer(environment.Merge(task.proto.Environment, env), boolevator).Schema(),
			func(node *node.Node) error {
				inst := instance.NewCommunityContainer(environment.Merge(task.proto.Environment, env), boolevator)
				containerInstance, err := inst.Parse(node)
				if err != nil {
					return err
				}

				// Retrieve the platform to update the environment
				task.proto.Environment = environment.Merge(
					task.proto.Environment,
					map[string]string{"CIRRUS_OS": strings.ToLower(containerInstance.Platform.String())},
				)

				anyInstance, err := ptypes.MarshalAny(containerInstance)
				if err != nil {
					return err
				}
				task.proto.Instance = anyInstance

				return nil
			})
	}
	if _, ok := additionalInstances["persistent_worker"]; !ok {
		task.CollectibleField("persistent_worker",
			instance.NewPersistentWorker().Schema(),
			func(node *node.Node) error {
				inst := instance.NewPersistentWorker()
				persistentWorkerInstance, err := inst.Parse(node)
				if err != nil {
					return err
				}

				anyInstance, err := ptypes.MarshalAny(persistentWorkerInstance)
				if err != nil {
					return err
				}
				task.proto.Instance = anyInstance

				return nil
			},
		)
	}

	for instanceName, descriptor := range additionalInstances {
		scopedInstanceName := instanceName
		scopedDescriptor := descriptor
		task.CollectibleField(scopedInstanceName,
			instance.NewProtoParser(scopedDescriptor, environment.Merge(task.proto.Environment, env), boolevator).Schema(),
			func(node *node.Node) error {
				parser := instance.NewProtoParser(scopedDescriptor, environment.Merge(task.proto.Environment, env), boolevator)
				parserInstance, err := parser.Parse(node)
				if err != nil {
					return err
				}
				anyInstance, err := anypb.New(parserInstance)
				if err != nil {
					return err
				}
				task.proto.Instance = anyInstance
				task.proto.Environment = environment.Merge(
					task.proto.Environment, map[string]string{
						"CIRRUS_OS": instance.GuessPlatform(anyInstance, scopedDescriptor),
					},
				)
				return nil
			})
	}

	task.OptionalField(nameable.NewSimpleNameable("name"), schema.String(""), func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(environment.Merge(task.proto.Environment, env))
		if err != nil {
			return err
		}
		task.proto.Name = name
		return nil
	})

	task.OptionalField(nameable.NewSimpleNameable("alias"), schema.String(""), func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(environment.Merge(task.proto.Environment, env))
		if err != nil {
			return err
		}
		task.alias = name
		return nil
	})

	bgNameable := nameable.NewRegexNameable("^(.*)background_script$")
	task.OptionalField(bgNameable, schema.Script(""), func(node *node.Node) error {
		command, err := handleBackgroundScript(node, bgNameable)
		if err != nil {
			return err
		}

		task.proto.Commands = append(task.proto.Commands, command)

		return nil
	})

	scriptNameable := nameable.NewRegexNameable("^(.*)script$")
	task.OptionalField(scriptNameable, schema.Script(""), func(node *node.Node) error {
		command, err := handleScript(node, scriptNameable)
		if err != nil {
			return err
		}

		task.proto.Commands = append(task.proto.Commands, command)

		return nil
	})

	cacheNameable := nameable.NewRegexNameable("^(.*)cache$")
	cacheSchema := command.NewCacheCommand(nil, nil).Schema()
	task.OptionalField(cacheNameable, cacheSchema, func(node *node.Node) error {
		cache := command.NewCacheCommand(environment.Merge(task.proto.Environment, env), boolevator)
		if err := cache.Parse(node); err != nil {
			return err
		}
		task.proto.Commands = append(task.proto.Commands, cache.Proto())
		return nil
	})

	artifactsNameable := nameable.NewRegexNameable("^(.*)artifacts$")
	artifactsSchema := command.NewArtifactsCommand(nil).Schema()
	task.OptionalField(artifactsNameable, artifactsSchema, func(node *node.Node) error {
		artifacts := command.NewArtifactsCommand(environment.Merge(task.proto.Environment, env))
		if err := artifacts.Parse(node); err != nil {
			return err
		}
		task.proto.Commands = append(task.proto.Commands, artifacts.Proto())
		return nil
	})

	fileNameable := nameable.NewRegexNameable("^(.*)file$")
	fileSchema := command.NewFileCommand(nil).Schema()
	task.OptionalField(fileNameable, fileSchema, func(node *node.Node) error {
		file := command.NewFileCommand(environment.Merge(task.proto.Environment, env))
		if err := file.Parse(node); err != nil {
			return err
		}
		task.proto.Commands = append(task.proto.Commands, file.Proto())
		return nil
	})

	task.registerExecutionBehaviorFields(env, boolevator)

	dependsOnSchema := schema.StringOrListOfStrings("List of task names this task depends on.")
	task.OptionalField(nameable.NewSimpleNameable("depends_on"), dependsOnSchema, func(node *node.Node) error {
		dependsOn, err := node.GetSliceOfNonEmptyStrings()
		if err != nil {
			return err
		}
		task.dependsOn = dependsOn
		return nil
	})

	task.CollectibleField("only_if", schema.Condition(""), func(node *node.Node) error {
		onlyIfExpression, err := node.GetStringValue()
		if err != nil {
			return err
		}
		task.onlyIfExpression = onlyIfExpression
		return nil
	})

	task.CollectibleField("skip", schema.Condition(""), func(node *node.Node) error {
		skipped, err := node.GetBoolValue(environment.Merge(task.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		if skipped {
			task.proto.Status = api.Task_SKIPPED
		}
		return nil
	})

	task.CollectibleField("allow_failures", schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(task.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		task.proto.Metadata.Properties["allow_failures"] = strconv.FormatBool(evaluation)
		return nil
	})

	// for cloud only
	task.CollectibleField("skip_notifications", schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(task.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		task.proto.Metadata.Properties["skip_notifications"] = strconv.FormatBool(evaluation)
		return nil
	})

	// for cloud only
	task.CollectibleField("auto_cancellation", schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(task.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		task.proto.Metadata.Properties["auto_cancellation"] = strconv.FormatBool(evaluation)
		return nil
	})

	// for cloud only
	task.CollectibleField("use_compute_credits", schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(task.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		task.proto.Metadata.Properties["use_compute_credits"] = strconv.FormatBool(evaluation)
		return nil
	})

	// for cloud only
	task.CollectibleField("stateful", schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(task.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		task.proto.Metadata.Properties["stateful"] = strconv.FormatBool(evaluation)
		return nil
	})

	// no-op
	labelsSchema := schema.StringOrListOfStrings("List of required labels on a PR.")
	task.OptionalField(nameable.NewSimpleNameable("required_pr_labels"), labelsSchema, func(node *node.Node) error {
		return nil
	})

	task.CollectibleField("experimental", schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(task.proto.Environment, env), boolevator)
		if err != nil {
			return err
		}
		task.proto.Metadata.Properties["experimental"] = strconv.FormatBool(evaluation)
		return nil
	})

	task.CollectibleField("timeout_in", schema.Number("Task timeout in minutes"), func(node *node.Node) error {
		timeout, err := handleTimeoutIn(node, environment.Merge(task.proto.Environment, env))
		if err != nil {
			return err
		}

		task.proto.Metadata.Properties["timeout_in"] = timeout

		return nil
	})

	task.CollectibleField("trigger_type", schema.TriggerType(), func(node *node.Node) error {
		triggerType, err := node.GetExpandedStringValue(environment.Merge(task.proto.Environment, env))
		if err != nil {
			return err
		}
		task.proto.Metadata.Properties["trigger_type"] = strings.ToUpper(triggerType)
		return nil
	})

	lockSchema := schema.String("Lock name for triggering and execution")
	task.OptionalField(nameable.NewSimpleNameable("execution_lock"), lockSchema, func(node *node.Node) error {
		lockName, err := node.GetExpandedStringValue(environment.Merge(task.proto.Environment, env))
		if err != nil {
			return err
		}

		task.proto.Metadata.Properties["execution_lock"] = lockName

		return nil
	})

	return task
}

func (task *Task) Parse(node *node.Node) error {
	if err := task.DefaultParser.Parse(node); err != nil {
		return err
	}

	if task.proto.Instance == nil {
		return fmt.Errorf("%w: task %s has no instance attached", parsererror.ErrParsing, task.Name())
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
}

func (task *Task) SetIndexWithinBuild(id int64) {
	task.proto.Metadata.Properties["indexWithinBuild"] = strconv.FormatInt(id, 10)
}

func (task *Task) DependsOnIDs() []int64       { return task.proto.RequiredGroups }
func (task *Task) SetDependsOnIDs(ids []int64) { task.proto.RequiredGroups = ids }

func (task *Task) Proto() interface{} {
	return &task.proto
}

func (task *Task) Enabled(env map[string]string, boolevator *boolevator.Boolevator) (bool, error) {
	if task.onlyIfExpression == "" {
		return true, nil
	}

	evaluation, err := boolevator.Eval(task.onlyIfExpression, environment.Merge(task.proto.Environment, env))
	if err != nil {
		return false, err
	}

	return evaluation, nil
}

func (task *Task) registerExecutionBehaviorFields(env map[string]string, boolevator *boolevator.Boolevator) {
	for id, name := range api.Command_CommandExecutionBehavior_name {
		idCopy := id

		behaviorSchema := NewBehavior(nil, nil).Schema()
		behaviorSchema.Description = name + " commands."
		task.OptionalField(nameable.NewSimpleNameable(strings.ToLower(name)), behaviorSchema, func(node *node.Node) error {
			behavior := NewBehavior(environment.Merge(task.proto.Environment, env), boolevator)
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

func (task *Task) Schema() *jsschema.Schema {
	modifiedSchema := task.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "Cirrus CI task definition."

	return modifiedSchema
}
