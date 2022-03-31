package parseable

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	nodepkg "github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parserkit"
	"github.com/lestrrat-go/jsschema"
	"regexp"
)

type Parseable interface {
	Parse(node *nodepkg.Node, parserKit *parserkit.ParserKit) error
	Schema() *schema.Schema
	CollectibleFields() []CollectibleField
	Proto() interface{}
}

type nodeFunc func(node *nodepkg.Node) error

type Field struct {
	name       nameable.Nameable
	required   bool
	repeatable bool
	onFound    nodeFunc
	schema     *schema.Schema
}

type CollectibleField struct {
	Name    string
	onFound nodeFunc
	Schema  *schema.Schema
}

func (parser *DefaultParser) Parse(node *nodepkg.Node, parserKit *parserkit.ParserKit) error {
	// Detect possible incorrect usage of fields that expect a map
	// (e.g. "container: ruby:latest"), yet allow "container:" since
	// there's no clear intention to configure this field from the user
	if _, ok := node.Value.(*nodepkg.MapValue); !ok && !node.ValueIsEmpty() {
		parserKit.IssueRegistry.RegisterIssuef(api.Issue_WARNING, node.Line, node.Column,
			"expected a map, found %s", node.ValueTypeAsString())
	}

	// Evaluate collectible fields
	for _, field := range parser.collectibleFields {
		if err := evaluateCollectible(node, field); err != nil {
			return err
		}
	}

	for _, child := range node.Children {
		// double check collectible fields
		for _, collectibleField := range parser.collectibleFields {
			if collectibleField.Name == child.Name {
				if err := evaluateCollectible(node, collectibleField); err != nil {
					return err
				}
				break
			}
		}
	}

	// Calculate redefinitions index to answer the question:
	// "Is the field we're processing right now will be redefined later?"
	redefinitions := map[string]*nodepkg.Node{}

	for _, child := range node.Children {
		redefinitions[child.Name] = child
	}

	// Evaluate ordinary fields
	seenFields := map[string]struct{}{}

	for _, child := range node.Children {
		field := parser.NormalField(child.Name)
		if field == nil {
			continue
		}

		redefinition, hasRedefinition := redefinitions[child.Name]
		redifinitionWillBeEncounteredLater := hasRedefinition && child != redefinition
		if !field.repeatable && redifinitionWillBeEncounteredLater {
			continue
		}

		seenFields[field.name.MapKey()] = struct{}{}

		if err := field.onFound(child); err != nil {
			return err
		}
	}

	// Check ordinary fields with the "required" flag set
	for _, field := range parser.fields {
		_, seen := seenFields[field.name.MapKey()]

		if field.required && !seen {
			return node.ParserError("required field %q was not set", field.name.String())
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

func evaluateCollectible(node *nodepkg.Node, field CollectibleField) error {
	children := node.DeepFindCollectible(field.Name)

	if children == nil {
		return nil
	}

	return field.onFound(children)
}
