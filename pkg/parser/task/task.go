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

func NewTask(
	env map[string]string,
	boolevator *boolevator.Boolevator,
	additionalInstances map[string]protoreflect.MessageDescriptor,
) *Task {
	task := &Task{}

	// Don't force required fields in schema
	task.SetCollectible(true)

	AttachBaseTaskFields(&task.DefaultParser, &task.proto, env, boolevator)

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

				// Clean CIRRUS_OS since we don't know where we will be running
				delete(task.proto.Environment, "CIRRUS_OS")

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

		instanceSchema := instance.NewProtoParser(scopedDescriptor, nil, nil).Schema()
		task.CollectibleField(scopedInstanceName, instanceSchema, func(node *node.Node) error {
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

	task.OptionalField(nameable.NewSimpleNameable("alias"), schema.String(""), func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(environment.Merge(task.proto.Environment, env))
		if err != nil {
			return err
		}
		task.alias = name
		return nil
	})

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

func (task *Task) Schema() *jsschema.Schema {
	modifiedSchema := task.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "Cirrus CI task definition."

	return modifiedSchema
}
