package tart

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parserkit"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	jsschema "github.com/lestrrat-go/jsschema"
)

type Volume struct {
	proto *api.Isolation_Tart_Volume

	parseable.DefaultParser
}

func NewVolume(mergedEnv map[string]string, parserKit *parserkit.ParserKit) *Volume {
	volume := &Volume{
		proto: &api.Isolation_Tart_Volume{},
	}

	nameSchema := schema.String("Volume name that will be mounted into /Volumes/My Shared Files/<name>.")
	volume.OptionalField(nameable.NewSimpleNameable("name"), nameSchema, func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		volume.proto.Name = name

		return nil
	})

	sourceSchema := schema.String("Volume source, a path to a directory on the host.")
	volume.RequiredField(nameable.NewSimpleNameable("source"), sourceSchema, func(node *node.Node) error {
		source, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		volume.proto.Source = source

		return nil
	})

	targetSchema := schema.String("Volume target, a path to a directory in the guest.")
	volume.OptionalField(nameable.NewSimpleNameable("target"), targetSchema, func(node *node.Node) error {
		target, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		volume.proto.Target = target

		return nil
	})

	readOnlySchema := schema.String("Whether this volume should be mounted in readonly mode.")
	volume.OptionalField(nameable.NewSimpleNameable("readonly"), readOnlySchema, func(node *node.Node) error {
		readOnly, err := node.GetBoolValue(mergedEnv, parserKit.Boolevator)
		if err != nil {
			return err
		}

		volume.proto.ReadOnly = readOnly

		return nil
	})

	return volume
}

func (volume *Volume) Parse(
	node *node.Node,
	parserKit *parserkit.ParserKit,
) (*api.Isolation_Tart_Volume, error) {
	if err := volume.DefaultParser.Parse(node, parserKit); err != nil {
		return nil, err
	}

	return volume.proto, nil
}

func (volume *Volume) Schema() *jsschema.Schema {
	modifiedSchema := volume.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "Tart volume definition."

	return modifiedSchema
}
