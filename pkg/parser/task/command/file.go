package command

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"strings"
)

type FileCommand struct {
	proto       *api.Command
	instruction *api.FileInstruction

	parseable.DefaultParser
}

func NewFileCommand(mergedEnv map[string]string) *FileCommand {
	fileCommand := &FileCommand{
		proto:       &api.Command{},
		instruction: &api.FileInstruction{},
	}

	fileCommand.OptionalField(nameable.NewSimpleNameable("name"), schema.TodoSchema, func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		fileCommand.proto.Name = name
		return nil
	})

	fileCommand.RequiredField(nameable.NewSimpleNameable("path"), schema.TodoSchema, func(node *node.Node) error {
		path, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		fileCommand.instruction.DestinationPath = path
		return nil
	})

	fileCommand.RequiredField(nameable.NewSimpleNameable("variable_name"), schema.TodoSchema, func(node *node.Node) error {
		variableName, err := node.GetStringValue()
		if err != nil {
			return err
		}

		variableName = strings.TrimPrefix(variableName, "$")

		fileCommand.instruction.Source = &api.FileInstruction_FromEnvironmentVariable{
			FromEnvironmentVariable: variableName,
		}

		return nil
	})

	return fileCommand
}

func (fileCommand *FileCommand) Parse(node *node.Node) error {
	if err := fileCommand.DefaultParser.Parse(node); err != nil {
		return err
	}

	if fileCommand.proto.Name == "" {
		cacheNameable := nameable.NewRegexNameable("^(.*)file$")
		fileCommand.proto.Name = cacheNameable.FirstGroupOrDefault(node.Name, "file")
	}

	return nil
}

func (fileCommand *FileCommand) Proto() *api.Command {
	fileCommand.proto.Instruction = &api.Command_FileInstruction{
		FileInstruction: fileCommand.instruction,
	}

	return fileCommand.proto
}
