package command

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
)

type ArtifactsCommand struct {
	proto       *api.Command
	instruction *api.ArtifactsInstruction

	parseable.DefaultParser
}

func NewArtifactsCommand(mergedEnv map[string]string) *ArtifactsCommand {
	articom := &ArtifactsCommand{
		proto:       &api.Command{},
		instruction: &api.ArtifactsInstruction{},
	}

	articom.OptionalField(nameable.NewSimpleNameable("paths"), schema.TodoSchema, func(node *node.Node) error {
		artifactPaths, err := node.GetSliceOfExpandedStrings(mergedEnv)
		if err != nil {
			return err
		}
		articom.instruction.Paths = artifactPaths
		return nil
	})

	articom.OptionalField(nameable.NewSimpleNameable("path"), schema.TodoSchema, func(node *node.Node) error {
		artifactPath, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		articom.instruction.Paths = []string{artifactPath}
		return nil
	})

	articom.OptionalField(nameable.NewSimpleNameable("type"), schema.TodoSchema, func(node *node.Node) error {
		artifactType, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		articom.instruction.Type = artifactType
		return nil
	})

	articom.OptionalField(nameable.NewSimpleNameable("format"), schema.TodoSchema, func(node *node.Node) error {
		artifactFormat, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		articom.instruction.Format = artifactFormat
		return nil
	})

	return articom
}

func (articom *ArtifactsCommand) Parse(node *node.Node) error {
	if err := articom.DefaultParser.Parse(node); err != nil {
		return err
	}

	cacheNameable := nameable.NewRegexNameable("^(.*)artifacts$")
	articom.proto.Name = cacheNameable.FirstGroupOrDefault(node.Name, "binary")

	return nil
}

func (articom *ArtifactsCommand) Proto() *api.Command {
	articom.proto.Instruction = &api.Command_ArtifactsInstruction{
		ArtifactsInstruction: articom.instruction,
	}

	return articom.proto
}
