package task

import (
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parserkit"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/task/command"
	jsschema "github.com/lestrrat-go/jsschema"
	"strings"
)

type PipeStep struct {
	protoCommands []*api.Command

	image string

	parseable.DefaultParser
}

func NewPipeStep(
	mergedEnv map[string]string,
	parserKit *parserkit.ParserKit,
	previousCommands []*api.Command,
) *PipeStep {
	step := &PipeStep{}

	imageSchema := schema.String("Docker Image to use.")
	step.RequiredField(nameable.NewSimpleNameable("image"), imageSchema, func(node *node.Node) error {
		image, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		step.image = image
		return nil
	})

	scriptNameable := nameable.NewRegexNameable("^(.*)script$")
	step.OptionalField(scriptNameable, schema.Script(""), func(node *node.Node) error {
		command, err := handleScript(node, scriptNameable)
		if err != nil {
			return err
		}

		step.protoCommands = append(step.protoCommands, command)

		return nil
	})

	cacheNameable := nameable.NewRegexNameable("^(.*)cache$")
	cacheSchema := command.NewCacheCommand(nil, nil).Schema()
	step.OptionalField(cacheNameable, cacheSchema, func(node *node.Node) error {
		cache := command.NewCacheCommand(mergedEnv, parserKit)
		if err := cache.Parse(node, parserKit); err != nil {
			return err
		}
		step.protoCommands = append(step.protoCommands, cache.Proto())
		return nil
	})

	uploadCachesNameable := nameable.NewSimpleNameable("upload_caches")
	step.OptionalRepeatableField(uploadCachesNameable, command.UploadCachesSchema(), func(node *node.Node) error {
		commandsToAppend, err := command.UploadCachesHelper(mergedEnv, append(previousCommands, step.protoCommands...), node)
		if err != nil {
			return err
		}

		step.protoCommands = append(step.protoCommands, commandsToAppend...)

		return nil
	})

	artifactsNameable := nameable.NewRegexNameable("^(.*)artifacts$")
	artifactsSchema := command.NewArtifactsCommand(nil).Schema()
	step.OptionalField(artifactsNameable, artifactsSchema, func(node *node.Node) error {
		artifacts := command.NewArtifactsCommand(mergedEnv)
		if err := artifacts.Parse(node, parserKit); err != nil {
			return err
		}
		step.protoCommands = append(step.protoCommands, artifacts.Proto())
		return nil
	})

	fileNameable := nameable.NewRegexNameable("^(.*)file$")
	fileSchema := command.NewFileCommand(nil).Schema()
	step.OptionalField(fileNameable, fileSchema, func(node *node.Node) error {
		file := command.NewFileCommand(mergedEnv)
		if err := file.Parse(node, parserKit); err != nil {
			return err
		}
		step.protoCommands = append(step.protoCommands, file.Proto())
		return nil
	})

	for id, name := range api.Command_CommandExecutionBehavior_name {
		idCopy := id

		behaviorName := nameable.NewSimpleNameable(strings.ToLower(name))
		behaviorSchema := NewBehavior(nil, nil, nil).Schema()
		behaviorSchema.Description = name + " commands."
		step.OptionalRepeatableField(behaviorName, behaviorSchema, func(node *node.Node) error {
			behavior := NewBehavior(mergedEnv, parserKit, append(previousCommands, step.protoCommands...))
			if err := behavior.Parse(node, parserKit); err != nil {
				return err
			}

			commands := behavior.Proto()

			for _, command := range commands {
				command.ExecutionBehaviour = api.Command_CommandExecutionBehavior(idCopy)
				step.protoCommands = append(step.protoCommands, command)
			}

			return nil
		})
	}

	return step
}

func (step *PipeStep) Parse(node *node.Node, parserKit *parserkit.ParserKit) error {
	if err := step.DefaultParser.Parse(node, parserKit); err != nil {
		return err
	}

	if len(step.protoCommands) == 0 {
		return node.ParserError("pipe steps defined without scripts inside them")
	}

	step.protoCommands[0].Properties = make(map[string]string)
	step.protoCommands[0].Properties["image"] = step.image

	return nil
}

func (step *PipeStep) Schema() *jsschema.Schema {
	modifiedSchema := step.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "Pipe step"

	return modifiedSchema
}
