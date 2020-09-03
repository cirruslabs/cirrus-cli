package task

import (
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"strings"
)

type PipeStep struct {
	protoCommands []*api.Command

	image string

	parseable.DefaultParser
}

func NewPipeStep() *PipeStep {
	step := &PipeStep{}

	step.RequiredField(nameable.NewSimpleNameable("image"), schema.TodoSchema, func(node *node.Node) error {
		image, err := node.GetExpandedStringValue(nil)
		if err != nil {
			return err
		}
		step.image = image
		return nil
	})

	scriptNameable := nameable.NewRegexNameable("(.*)script")
	step.OptionalField(scriptNameable, schema.TodoSchema, func(node *node.Node) error {
		scripts, err := node.GetSliceOfNonEmptyStrings()
		if err != nil {
			return err
		}
		step.protoCommands = append(step.protoCommands, &api.Command{
			Name: scriptNameable.FirstGroupOrDefault(node.Name, "main"),
			Instruction: &api.Command_ScriptInstruction{
				ScriptInstruction: &api.ScriptInstruction{
					Scripts: scripts,
				},
			},
		})
		return nil
	})

	for id, name := range api.Command_CommandExecutionBehavior_name {
		idCopy := id
		step.OptionalField(nameable.NewSimpleNameable(strings.ToLower(name)), schema.TodoSchema, func(node *node.Node) error {
			behavior := NewBehavior()
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
