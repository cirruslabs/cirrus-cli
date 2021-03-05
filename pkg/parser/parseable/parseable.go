package parseable

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/lestrrat-go/jsschema"
	"regexp"
	"sort"
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
	Name            string
	onFound         nodeFunc
	Schema          *schema.Schema
	DefinesInstance bool
}

func (parser *DefaultParser) Parse(node *node.Node) error {
	// Rank collectible fields according to their declaration order in the node, e.g.:
	//
	// task:
	//   container:
	//     ...
	//   windows_container:
	//     ...
	//
	// ...results in:
	//
	// map[string]int{
	//   {"container": 1},
	//   {"windows_container": 2},
	// }
	//
	// Note that the first field starts with 1, leaving 0 for the fields that we haven't seen
	// in the node to prevent collisions.
	fieldPositions := map[string]int{}
	for i, child := range node.Children {
		fieldPositions[child.Name] = i + 1
	}

	// Sort collectible fields:
	//
	// * unspecified fields (fields with rank 0) first
	// * fields specified in the node that _don't_ describe instances (in the reverse order of declaration)
	// * fields specified in the node that describe instances (in the reverse order of declaration)
	//
	// ...to achieve the (1) deprioritize, (2) overwritten-by-the-user and (2) instances-evaluated-at-the-end properties,
	// where:
	//
	// (1) means that an instance (e.g. container) defined at the task's scope should be preferred
	//     to the instance defined at the root scope (also container)
	//
	// (2) means that if two instances (e.g. container first and the persistent_worker) are defined at the same level,
	//     the persistent_worker instance overwrites the container instance
	//
	// (3) means that collectible fields that don't define instances (e.g. "env") are evaluated first,
	//     because instances might use environment variables defined in such fields
	rankedCollectibles := parser.collectibleFields

	sort.Slice(rankedCollectibles, func(i, j int) bool {
		iField := rankedCollectibles[i]
		jField := rankedCollectibles[j]

		// Golang docs state that:
		// >Less reports whether x[i] should be ordered before x[j], as required by the sort Interface.

		// iField should be ordered before jField if it comes first in the node fields
		// (or wasn't defined at all)
		if iField.DefinesInstance && jField.DefinesInstance {
			return fieldPositions[iField.Name] < fieldPositions[jField.Name]
		}

		// iField should be ordered before jField because if it does not define an instance
		return !iField.DefinesInstance
	})

	// Evaluate collectible fields
	for _, field := range rankedCollectibles {
		if err := evaluateCollectible(node, field); err != nil {
			return err
		}
	}

	// Check required fields

	for _, child := range node.Children {
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

	return field.onFound(children)
}
