package task

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/task/command"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"google.golang.org/protobuf/types/descriptorpb"
	"strconv"
	"strings"
)

// nolint:gocognit // it's a parser helper, there is a lot of boilerplate
func AttachBaseTaskFields(
	parser *parseable.DefaultParser,
	task *api.Task,
	env map[string]string,
	boolevator *boolevator.Boolevator,
	additionalTaskProperties []*descriptor.FieldDescriptorProto,
) {
	task.Metadata = &api.Task_Metadata{Properties: DefaultTaskProperties()}

	autoCancellation := env["CIRRUS_BRANCH"] != env["CIRRUS_DEFAULT_BRANCH"]
	if autoCancellation {
		task.Metadata.Properties["auto_cancellation"] = strconv.FormatBool(autoCancellation)
	}

	parser.CollectibleField("environment", schema.Map(""), func(node *node.Node) error {
		taskEnv, err := node.GetEnvironment()
		if err != nil {
			return err
		}
		task.Environment = environment.Merge(task.Environment, taskEnv)
		return nil
	})

	parser.CollectibleField("env", schema.Map(""), func(node *node.Node) error {
		taskEnv, err := node.GetEnvironment()
		if err != nil {
			return err
		}
		task.Environment = environment.Merge(task.Environment, taskEnv)
		return nil
	})

	parser.OptionalField(nameable.NewSimpleNameable("name"), schema.String(""), func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(environment.Merge(task.Environment, env))
		if err != nil {
			return err
		}
		task.Name = name
		return nil
	})

	parser.CollectibleField("skip", schema.Condition(""), func(node *node.Node) error {
		skipped, err := node.GetBoolValue(environment.Merge(task.Environment, env), boolevator)
		if err != nil {
			return err
		}
		if skipped {
			task.Status = api.Task_SKIPPED
		}
		return nil
	})

	parser.CollectibleField("allow_failures", schema.Condition(""), func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(environment.Merge(task.Environment, env), boolevator)
		if err != nil {
			return err
		}
		task.Metadata.Properties["allow_failures"] = strconv.FormatBool(evaluation)
		return nil
	})

	parser.CollectibleField("timeout_in", schema.Number("Task timeout in minutes"), func(node *node.Node) error {
		timeout, err := handleTimeoutIn(node, environment.Merge(task.Environment, env))
		if err != nil {
			return err
		}

		task.Metadata.Properties["timeout_in"] = timeout

		return nil
	})

	for _, additionalTaskProperty := range additionalTaskProperties {
		fieldNamePtr := additionalTaskProperty.Name
		fieldTypePtr := additionalTaskProperty.Type
		if fieldNamePtr == nil || fieldTypePtr == nil {
			continue
		}
		fieldName := *fieldNamePtr
		fieldType := *fieldTypePtr
		switch fieldType {
		case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
			parser.CollectibleField(fieldName, schema.Condition(""), func(node *node.Node) error {
				evaluation, err := node.GetBoolValue(environment.Merge(task.Environment, env), boolevator)
				if err != nil {
					return err
				}
				task.Metadata.Properties[fieldName] = strconv.FormatBool(evaluation)
				return nil
			})
		case descriptorpb.FieldDescriptorProto_TYPE_GROUP:
			parser.CollectibleField(fieldName, schema.StringOrListOfStrings(""), func(node *node.Node) error {
				evaluation, err := node.GetSliceOfExpandedStrings(environment.Merge(task.Environment, env))
				if err != nil {
					return err
				}
				task.Metadata.Properties[fieldName] = strings.Join(evaluation, "\n")
				return nil
			})
		default:
			parser.CollectibleField(fieldName, schema.String(""), func(node *node.Node) error {
				evaluation, err := node.GetExpandedStringValue(environment.Merge(task.Environment, env))
				if err != nil {
					return err
				}
				task.Metadata.Properties[fieldName] = evaluation
				return nil
			})
		}
	}
}

func AttachBaseTaskInstructions(
	parser *parseable.DefaultParser,
	task *api.Task,
	env map[string]string,
	boolevator *boolevator.Boolevator,
) {
	bgNameable := nameable.NewRegexNameable("^(.*)background_script$")
	parser.OptionalField(bgNameable, schema.Script(""), func(node *node.Node) error {
		command, err := handleBackgroundScript(node, bgNameable)
		if err != nil {
			return err
		}

		task.Commands = append(task.Commands, command)

		return nil
	})

	scriptNameable := nameable.NewRegexNameable("^(.*)script$")
	parser.OptionalField(scriptNameable, schema.Script(""), func(node *node.Node) error {
		command, err := handleScript(node, scriptNameable)
		if err != nil {
			return err
		}

		task.Commands = append(task.Commands, command)

		return nil
	})

	cacheNameable := nameable.NewRegexNameable("^(.*)cache$")
	cacheSchema := command.NewCacheCommand(nil, nil).Schema()
	parser.OptionalField(cacheNameable, cacheSchema, func(node *node.Node) error {
		cache := command.NewCacheCommand(environment.Merge(task.Environment, env), boolevator)
		if err := cache.Parse(node); err != nil {
			return err
		}
		task.Commands = append(task.Commands, cache.Proto())
		return nil
	})

	artifactsNameable := nameable.NewRegexNameable("^(.*)artifacts$")
	artifactsSchema := command.NewArtifactsCommand(nil).Schema()
	parser.OptionalField(artifactsNameable, artifactsSchema, func(node *node.Node) error {
		artifacts := command.NewArtifactsCommand(environment.Merge(task.Environment, env))
		if err := artifacts.Parse(node); err != nil {
			return err
		}
		task.Commands = append(task.Commands, artifacts.Proto())
		return nil
	})

	fileNameable := nameable.NewRegexNameable("^(.*)file$")
	fileSchema := command.NewFileCommand(nil).Schema()
	parser.OptionalField(fileNameable, fileSchema, func(node *node.Node) error {
		file := command.NewFileCommand(environment.Merge(task.Environment, env))
		if err := file.Parse(node); err != nil {
			return err
		}
		task.Commands = append(task.Commands, file.Proto())
		return nil
	})

	for id, name := range api.Command_CommandExecutionBehavior_name {
		idCopy := id

		behaviorSchema := NewBehavior(nil, nil).Schema()
		behaviorSchema.Description = name + " commands."
		parser.OptionalField(nameable.NewSimpleNameable(strings.ToLower(name)), behaviorSchema, func(node *node.Node) error {
			behavior := NewBehavior(environment.Merge(task.Environment, env), boolevator)
			if err := behavior.Parse(node); err != nil {
				return err
			}

			commands := behavior.Proto()

			for _, command := range commands {
				command.ExecutionBehaviour = api.Command_CommandExecutionBehavior(idCopy)
				task.Commands = append(task.Commands, command)
			}

			return nil
		})
	}
}
