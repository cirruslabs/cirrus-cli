package task

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/constants"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parserkit"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/task/command"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	jsschema "github.com/lestrrat-go/jsschema"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"strconv"
	"strings"
)

type Task struct {
	proto api.Task

	instanceNode *node.Node

	fallbackName string
	alias        string
	dependsOn    []string

	onlyIfExpression string

	missingInstancesAllowed bool
	line                    int
	column                  int

	parseable.DefaultParser
}

//nolint:gocognit,nestif // it's a parser helper, there is a lot of boilerplate
func NewTask(
	env map[string]string,
	parserKit *parserkit.ParserKit,
	additionalInstances map[string]protoreflect.MessageDescriptor,
	additionalTaskProperties []*descriptor.FieldDescriptorProto,
	missingInstancesAllowed bool,
	line int,
	column int,
) *Task {
	task := &Task{
		missingInstancesAllowed: missingInstancesAllowed,
		line:                    line,
		column:                  column,
	}

	// Don't force required fields in schema
	task.SetCollectible(true)

	// Make sure environment is parsed first and then all the possible instance
	// because an instance can set CIRRUS_OS which can be used in some field (for example, "name: Tests ($CIRRUS_OS)").
	AttachEnvironmentFields(&task.DefaultParser, &task.proto)

	if _, ok := additionalInstances["container"]; !ok {
		task.CollectibleField("container",
			instance.NewCommunityContainer(environment.Merge(task.proto.Environment, env), parserKit).Schema(),
			func(node *node.Node) error {
				task.instanceNode = node

				inst := instance.NewCommunityContainer(environment.Merge(task.proto.Environment, env), parserKit)
				containerInstance, err := inst.Parse(node, parserKit)
				if err != nil {
					return err
				}

				// Retrieve the platform to update the environment
				task.proto.Environment = environment.Merge(
					task.proto.Environment,
					map[string]string{"CIRRUS_OS": strings.ToLower(containerInstance.Platform.String())},
				)

				anyInstance, err := anypb.New(containerInstance)
				if err != nil {
					return err
				}
				task.proto.Instance = anyInstance

				return nil
			})
	}
	if _, ok := additionalInstances["windows_container"]; !ok {
		task.CollectibleField("windows_container",
			instance.NewWindowsCommunityContainer(environment.Merge(task.proto.Environment, env), parserKit).Schema(),
			func(node *node.Node) error {
				task.instanceNode = node

				inst := instance.NewWindowsCommunityContainer(environment.Merge(task.proto.Environment, env), parserKit)
				containerInstance, err := inst.Parse(node, parserKit)
				if err != nil {
					return err
				}

				// Retrieve the platform to update the environment
				task.proto.Environment = environment.Merge(
					task.proto.Environment,
					map[string]string{"CIRRUS_OS": strings.ToLower(containerInstance.Platform.String())},
				)

				anyInstance, err := anypb.New(containerInstance)
				if err != nil {
					return err
				}
				task.proto.Instance = anyInstance

				return nil
			})
	}
	if _, ok := additionalInstances["macos_instance"]; !ok {
		task.CollectibleField("macos_instance",
			instance.NewMacOSInstance(environment.Merge(task.proto.Environment, env), parserKit).Schema(),
			func(node *node.Node) error {
				task.instanceNode = node

				inst := instance.NewMacOSInstance(environment.Merge(task.proto.Environment, env), parserKit)
				macosInstance, err := inst.Parse(node, parserKit)
				if err != nil {
					return err
				}

				// Retrieve the platform to update the environment
				task.proto.Environment = environment.Merge(
					task.proto.Environment,
					map[string]string{"CIRRUS_OS": "darwin"},
				)

				anyInstance, err := anypb.New(macosInstance)
				if err != nil {
					return err
				}
				task.proto.Instance = anyInstance

				return nil
			})
	}
	if _, ok := additionalInstances["persistent_worker"]; !ok {
		task.CollectibleField("persistent_worker",
			instance.NewPersistentWorker(environment.Merge(task.proto.Environment, env), parserKit).Schema(),
			func(node *node.Node) error {
				task.instanceNode = node

				inst := instance.NewPersistentWorker(environment.Merge(task.proto.Environment, env), parserKit)
				persistentWorkerInstance, err := inst.Parse(node, parserKit)
				if err != nil {
					return err
				}

				if isolation := persistentWorkerInstance.Isolation; isolation != nil {
					isolationVars := map[string]string{}

					switch iso := isolation.Type.(type) {
					case *api.Isolation_Parallels_:
						if iso.Parallels != nil {
							isolationVars["CIRRUS_OS"] = strings.ToLower(iso.Parallels.Platform.String())
						}
					case *api.Isolation_Tart_:
						isolationVars["CIRRUS_OS"] = "darwin"
					}

					task.proto.Environment = environment.Merge(task.proto.Environment, isolationVars)
				} else {
					// Clear CIRRUS_OS since we don't know where we will be running
					delete(task.proto.Environment, "CIRRUS_OS")
				}

				anyInstance, err := anypb.New(persistentWorkerInstance)
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
			task.instanceNode = node

			parser := instance.NewProtoParser(scopedDescriptor, environment.Merge(task.proto.Environment, env), parserKit)
			parserInstance, err := parser.Parse(node, parserKit)
			if err != nil {
				return err
			}
			anyInstance, err := anypb.New(parserInstance)
			if err != nil {
				return err
			}
			task.proto.Instance = anyInstance
			platform := instance.GuessPlatform(anyInstance, scopedDescriptor)
			if platform != "" {
				task.proto.Environment = environment.Merge(
					task.proto.Environment, map[string]string{
						"CIRRUS_OS": platform,
					},
				)
			}
			architecture := instance.GuessArchitectureOfProtoMessage(anyInstance, scopedDescriptor)
			if architecture != "" {
				task.proto.Environment = environment.Merge(
					task.proto.Environment, map[string]string{
						constants.EnvironmentCirrusArch: architecture,
					},
				)
			}
			return nil
		})
	}

	// Only after environment and instances should we add all the rest fields.
	AttachBaseTaskFields(&task.DefaultParser, &task.proto, env, parserKit, additionalTaskProperties)
	AttachBaseTaskInstructions(&task.DefaultParser, &task.proto, env, parserKit)

	task.OptionalField(nameable.NewSimpleNameable("alias"), schema.String(""), func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(environment.Merge(task.proto.Environment, env))
		if err != nil {
			return err
		}

		task.alias = name
		task.proto.Metadata.Properties["alias"] = name

		return nil
	})

	dependsOnSchema := schema.StringOrListOfStrings("List of task names this task depends on.")
	task.OptionalField(nameable.NewSimpleNameable("depends_on"), dependsOnSchema, func(node *node.Node) error {
		dependsOn, err := node.GetSliceOfExpandedStrings(environment.Merge(task.proto.Environment, env))
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

func (task *Task) Parse(node *node.Node, parserKit *parserkit.ParserKit) error {
	if err := task.DefaultParser.Parse(node, parserKit); err != nil {
		return err
	}

	if task.proto.Instance == nil && !task.missingInstancesAllowed {
		return node.ParserError("task has no instance or container attached, " +
			"consult the documentation to find all possible execution environment types")
	}

	// Since the parsing is almost done and no other commands are expected,
	// we can safely append cache upload commands, if applicable
	task.proto.Commands = append(task.proto.Commands, command.GenUploadCacheCmds(task.proto.Commands)...)

	return nil
}

func (task *Task) Name() string {
	return task.proto.Name
}

func (task *Task) SetName(name string) {
	task.proto.Name = name
}

func (task *Task) FallbackName() string {
	return task.fallbackName
}

func (task *Task) SetFallbackName(name string) {
	task.fallbackName = name
}

func (task *Task) Alias() string {
	return task.alias
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

func (task *Task) InstanceNode() *node.Node {
	return task.instanceNode
}

func (task *Task) OnlyIfExpression() string {
	return task.onlyIfExpression
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

func (task *Task) Line() int {
	return task.line
}

func (task *Task) Column() int {
	return task.column
}

func (task *Task) Schema() *jsschema.Schema {
	modifiedSchema := task.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "Cirrus CI task definition."

	return modifiedSchema
}
