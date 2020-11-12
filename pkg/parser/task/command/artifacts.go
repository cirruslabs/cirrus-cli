package command

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	jsschema "github.com/lestrrat-go/jsschema"
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

	articom.OptionalField(nameable.NewSimpleNameable("name"), schema.String(""), func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		articom.proto.Name = name
		return nil
	})

	pathsSchema := schema.ArrayOf(schema.String("Path or pattern of artifacts."))
	articom.OptionalField(nameable.NewSimpleNameable("paths"), pathsSchema, func(node *node.Node) error {
		artifactPaths, err := node.GetSliceOfExpandedStrings(mergedEnv)
		if err != nil {
			return err
		}
		articom.instruction.Paths = artifactPaths
		return nil
	})

	pathSchema := schema.String("Path or pattern of artifacts.")
	articom.OptionalField(nameable.NewSimpleNameable("path"), pathSchema, func(node *node.Node) error {
		artifactPath, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		articom.instruction.Paths = []string{artifactPath}
		return nil
	})

	typeSchema := schema.String("Content Type.")
	articom.OptionalField(nameable.NewSimpleNameable("type"), typeSchema, func(node *node.Node) error {
		artifactType, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		articom.instruction.Type = artifactType
		return nil
	})

	formatSchema := schema.String("Content Format.")
	articom.OptionalField(nameable.NewSimpleNameable("format"), formatSchema, func(node *node.Node) error {
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

	if articom.proto.Name == "" {
		cacheNameable := nameable.NewRegexNameable("^(.*)artifacts$")
		articom.proto.Name = cacheNameable.FirstGroupOrDefault(node.Name, "binary")
	}

	return nil
}

func (articom *ArtifactsCommand) Proto() *api.Command {
	articom.proto.Instruction = &api.Command_ArtifactsInstruction{
		ArtifactsInstruction: articom.instruction,
	}

	return articom.proto
}

func (articom *ArtifactsCommand) Schema() *jsschema.Schema {
	modifiedSchema := articom.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}

	return modifiedSchema
}
