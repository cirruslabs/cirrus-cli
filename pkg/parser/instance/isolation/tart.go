package isolation

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parserkit"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	jsschema "github.com/lestrrat-go/jsschema"
)

type Tart struct {
	proto *api.Isolation_Tart_

	parseable.DefaultParser
}

func NewTart(mergedEnv map[string]string) *Tart {
	tart := &Tart{
		proto: &api.Isolation_Tart_{
			Tart: &api.Isolation_Tart{},
		},
	}

	vmSchema := schema.String("VM name.")
	tart.OptionalField(nameable.NewSimpleNameable("vm"), vmSchema, func(node *node.Node) error {
		vm, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		tart.proto.Tart.Vm = vm

		return nil
	})

	userSchema := schema.String("SSH username")
	tart.OptionalField(nameable.NewSimpleNameable("user"), userSchema, func(node *node.Node) error {
		user, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		tart.proto.Tart.User = user

		return nil
	})

	passwordSchema := schema.String("SSH password")
	tart.OptionalField(nameable.NewSimpleNameable("password"), passwordSchema, func(node *node.Node) error {
		password, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		tart.proto.Tart.Password = password

		return nil
	})

	return tart
}

func (tart *Tart) Parse(node *node.Node, parserKit *parserkit.ParserKit) error {
	return tart.DefaultParser.Parse(node, parserKit)
}

func (tart *Tart) Proto() *api.Isolation_Tart_ {
	return tart.proto
}

func (tart *Tart) Schema() *jsschema.Schema {
	modifiedSchema := tart.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "Tart VM isolation."

	return modifiedSchema
}
