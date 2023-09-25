package isolation

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance/isolation/tart"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parserkit"
	jsschema "github.com/lestrrat-go/jsschema"
)

type Isolation struct {
	proto api.Isolation

	parseable.DefaultParser
}

func NewIsolation(mergedEnv map[string]string, parserKit *parserkit.ParserKit) *Isolation {
	isolation := &Isolation{}

	parallelsSchema := NewParallels(mergedEnv).Schema()
	isolation.OptionalField(nameable.NewSimpleNameable("parallels"), parallelsSchema, func(node *node.Node) error {
		parallels := NewParallels(mergedEnv)

		if err := parallels.Parse(node, parserKit); err != nil {
			return err
		}

		isolation.proto.Type = parallels.Proto()

		return nil
	})

	containerSchema := NewContainer(mergedEnv).Schema()
	isolation.OptionalField(nameable.NewSimpleNameable("container"), containerSchema, func(node *node.Node) error {
		container := NewContainer(mergedEnv)

		if err := container.Parse(node, parserKit); err != nil {
			return err
		}

		isolation.proto.Type = container.Proto()

		return nil
	})

	tartSchema := tart.NewTart(mergedEnv, parserKit).Schema()
	isolation.OptionalField(nameable.NewSimpleNameable("tart"), tartSchema, func(node *node.Node) error {
		tart := tart.NewTart(mergedEnv, parserKit)

		if err := tart.Parse(node, parserKit); err != nil {
			return err
		}

		isolation.proto.Type = tart.Proto()

		return nil
	})

	return isolation
}

func (isolation *Isolation) Parse(node *node.Node, parserKit *parserkit.ParserKit) error {
	return isolation.DefaultParser.Parse(node, parserKit)
}

func (isolation *Isolation) Proto() *api.Isolation {
	return &isolation.proto
}

func (isolation *Isolation) Schema() *jsschema.Schema {
	modifiedSchema := isolation.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "Persistent Worker isolation."

	return modifiedSchema
}
