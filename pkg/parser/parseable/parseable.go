package parseable

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/lestrrat-go/jsschema"
)

type Parseable interface {
	Parse(node *node.Node) error
	Schema() *schema.Schema
	Proto() interface{}
}

type nodeFunc func(node *node.Node) error

type Field struct {
	name     nameable.Nameable
	required bool
	onFound  nodeFunc
	schema   *schema.Schema
}

type CollectibleField struct {
	name    string
	onFound nodeFunc
	schema  *schema.Schema
}

func (collectible *DefaultParser) Parse(node *node.Node) error {
	for _, field := range collectible.collectibleFields {
		children := node.DeepFindChildren(field.name)

		if children == nil {
			continue
		}

		if err := field.onFound(children); err != nil {
			return err
		}
	}

	// Check required fields

	for _, child := range node.Children {
		for _, field := range collectible.fields {
			if field.name.Matches(child.Name) {
				if err := field.onFound(child); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (collectible *DefaultParser) Schema() *schema.Schema {
	return &schema.Schema{}
}
