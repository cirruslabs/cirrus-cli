package parseable

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/lestrrat-go/jsschema"
	"regexp"
)

type Parseable interface {
	Parse(node *node.Node) error
	Schema() *schema.Schema
	CollectibleFields() []CollectibleField
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
	Name    string
	onFound nodeFunc
	Schema  *schema.Schema
}

func (parser *DefaultParser) Parse(node *node.Node) error {
	for _, field := range parser.collectibleFields {
		if err := evaluateCollectible(node, field); err != nil {
			return err
		}
	}

	// Check required fields

ChildrenLoop:
	for _, child := range node.Children {
		// In case this is a collectible field, make an additional evaluation
		// to guarantee that we follow the declaration order of instances
		// (and thus their overwrite order)
		for _, field := range parser.collectibleFields {
			if field.Name != child.Name {
				continue
			}

			if err := evaluateCollectible(node, field); err != nil {
				return err
			}

			continue ChildrenLoop
		}

		// Avoid processing the same node by different field handlers
		// and possibly generating multiple scripts from a single
		// script field
		var firstMatchedField *Field
		for i := range parser.fields {
			if parser.fields[i].name.Matches(child.Name) {
				firstMatchedField = &parser.fields[i]
				break
			}
		}

		if firstMatchedField == nil {
			continue
		}

		if err := firstMatchedField.onFound(child); err != nil {
			return err
		}
	}

	return nil
}

func (parser *DefaultParser) Schema() *schema.Schema {
	schema := &schema.Schema{
		Properties:           make(map[string]*schema.Schema),
		PatternProperties:    make(map[*regexp.Regexp]*schema.Schema),
		AdditionalItems:      &schema.AdditionalItems{Schema: nil},
		AdditionalProperties: &schema.AdditionalProperties{Schema: nil},
	}

	for _, field := range parser.fields {
		switch nameable := field.name.(type) {
		case *nameable.SimpleNameable:
			schema.Properties[nameable.Name()] = field.schema
		case *nameable.RegexNameable:
			schema.PatternProperties[nameable.Regex()] = field.schema
		}

		if field.required && !parser.Collectible() {
			schema.Required = append(schema.Required, field.name.String())
		}
	}

	for _, collectibleField := range parser.collectibleFields {
		schema.Properties[collectibleField.Name] = collectibleField.Schema
	}

	return schema
}

func (parser *DefaultParser) CollectibleFields() []CollectibleField {
	return parser.collectibleFields
}

func evaluateCollectible(node *node.Node, field CollectibleField) error {
	children := node.DeepFindCollectible(field.Name)

	if children == nil {
		return nil
	}

	if err := field.onFound(children); err != nil {
		return err
	}

	return nil
}
