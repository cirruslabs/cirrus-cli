package task

import (
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/task/command"
	"strings"
)

type PipeStep struct {
	protoCommands []*api.Command

	image string

	parseable.DefaultParser
}

func NewPipeStep(mergedEnv map[string]string) *PipeStep {
	step := &PipeStep{}

	step.RequiredField(nameable.NewSimpleNameable("image"), schema.TodoSchema, func(node *node.Node) error {
		image, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		step.image = image
		return nil
	})

	scriptNameable := nameable.NewRegexNameable("^(.*)script$")
	step.OptionalField(scriptNameable, schema.TodoSchema, func(node *node.Node) error {
		command, err := handleScript(node, scriptNameable)
		if err != nil {
			return err
		}

		step.protoCommands = append(step.protoCommands, command)

		return nil
	})

	cacheNameable := nameable.NewRegexNameable("^(.*)cache$")
	step.OptionalField(cacheNameable, schema.TodoSchema, func(node *node.Node) error {
		cache := NewCacheCommand(mergedEnv)
		if err := cache.Parse(node); err != nil {
			return err
		}
		step.protoCommands = append(step.protoCommands, cache.Proto())
		return nil
	})

	artifactsNameable := nameable.NewRegexNameable("^(.*)artifacts$")
	step.OptionalField(artifactsNameable, schema.TodoSchema, func(node *node.Node) error {
		artifacts := command.NewArtifactsCommand(mergedEnv)
		if err := artifacts.Parse(node); err != nil {
			return err
		}
		step.protoCommands = append(step.protoCommands, artifacts.Proto())
		return nil
	})

	for id, name := range api.Command_CommandExecutionBehavior_name {
		idCopy := id
		step.OptionalField(nameable.NewSimpleNameable(strings.ToLower(name)), schema.TodoSchema, func(node *node.Node) error {
			behavior := NewBehavior(mergedEnv)
			if err := behavior.Parse(node); err != nil {
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

func (step *PipeStep) Parse(node *node.Node) error {
	if err := step.DefaultParser.Parse(node); err != nil {
		return err
	}

	if len(step.protoCommands) == 0 {
		return fmt.Errorf("%w: there are pipe some steps defined without scripts inside them",
			parsererror.ErrParsing)
	}

	step.protoCommands[0].Properties = make(map[string]string)
	step.protoCommands[0].Properties["image"] = step.image

	return nil
}
