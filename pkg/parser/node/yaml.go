package node

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/cirruslabs/cirrus-cli/pkg/yamlhelper"
	"gopkg.in/yaml.v3"
)

var ErrFailedToMarshal = errors.New("failed to marshal to YAML")

type config struct {
	withoutYAMLNode bool
}

func (node *Node) MarshalYAML() (*yaml.Node, error) {
	switch obj := node.Value.(type) {
	case *MapValue:
		var resultChildren []*yaml.Node
		for _, child := range node.Children {
			marshalledItem, err := child.MarshalYAML()
			if err != nil {
				return nil, err
			}

			resultChildren = append(resultChildren, yamlhelper.NewStringNode(child.Name))
			resultChildren = append(resultChildren, marshalledItem)
		}

		return yamlhelper.NewMapNode(resultChildren), nil
	case *ListValue:
		var resultChildren []*yaml.Node
		for _, child := range node.Children {
			marshalledItem, err := child.MarshalYAML()
			if err != nil {
				return nil, err
			}

			resultChildren = append(resultChildren, marshalledItem)
		}

		return yamlhelper.NewSeqNode(resultChildren), nil
	case *ScalarValue:
		return yamlhelper.NewStringNode(obj.Value), nil
	default:
		return nil, fmt.Errorf("%w: unknown node type: %T", ErrFailedToMarshal, node.Value)
	}
}

func NewFromText(text string, opts ...Option) (*Node, error) {
	return NewFromTextWithMergeExemptions(text, []nameable.Nameable{}, opts...)
}

func NewFromTextWithMergeExemptions(text string, mergeExemptions []nameable.Nameable, opts ...Option) (*Node, error) {
	var yamlNode yaml.Node

	// Unmarshal YAML
	if err := yaml.Unmarshal([]byte(text), &yamlNode); err != nil {
		return nil, err
	}

	return NewFromNodeWithMergeExemptions(yamlNode, mergeExemptions, opts...)
}

func NewFromNodeWithMergeExemptions(
	yamlNode yaml.Node,
	mergeExemptions []nameable.Nameable,
	opts ...Option,
) (*Node, error) {
	// Empty document
	if yamlNode.Kind == 0 || len(yamlNode.Content) == 0 {
		return &Node{Name: "root"}, nil
	}

	if yamlNode.Kind != yaml.DocumentNode {
		return nil, parsererror.NewRich(yamlNode.Line, yamlNode.Column, "expected a YAML document, but got %s",
			yamlKindToString(yamlNode.Kind))
	}

	if len(yamlNode.Content) > 1 {
		extraneous := yamlNode.Content[1]

		return nil, parsererror.NewRich(yamlNode.Line, yamlNode.Column, "YAML document contains extraneous"+
			" top-level elements, such as %s", yamlKindToString(extraneous.Kind))
	}

	if yamlNode.Content[0].Kind != yaml.MappingNode {
		return nil, parsererror.NewRich(yamlNode.Line, yamlNode.Column, "YAML document should contain a mapping"+
			" as it top-level element, but found %s", yamlKindToString(yamlNode.Kind))
	}

	config := &config{}

	for _, opt := range opts {
		opt(config)
	}

	return convert(config, nil, "root", yamlNode.Content[0], yamlNode.Line, yamlNode.Column, mergeExemptions)
}

//nolint:gocognit // splitting this into multiple functions would probably make this even less comprehensible
func convert(
	config *config,
	parent *Node,
	name string,
	yamlNode *yaml.Node,
	line int,
	column int,
	mergeExemptions []nameable.Nameable,
) (*Node, error) {
	result := &Node{
		Name:   name,
		Parent: parent,
		Line:   line,
		Column: column,
	}

	if !config.withoutYAMLNode {
		result.YAMLNode = yamlNode
	}

	switch yamlNode.Kind {
	case yaml.SequenceNode:
		result.Value = &ListValue{}

		for _, item := range yamlNode.Content {
			// Ignore null[1] array elements for backwards compatibility
			//
			// [1]: https://yaml.org/type/null.html
			if item.Tag == "!!null" {
				continue
			}

			listSubtree, err := convert(config, result, "", item, item.Line, item.Column, mergeExemptions)
			if err != nil {
				return nil, err
			}

			result.Children = append(result.Children, listSubtree)
		}
	case yaml.MappingNode:
		result.Value = &MapValue{}

		if !isEven(len(yamlNode.Content)) {
			return nil, parsererror.NewRich(yamlNode.Line, yamlNode.Column, "unbalanced map")
		}

		for i := 0; i < len(yamlNode.Content); i += 2 {
			key := yamlNode.Content[i]
			value := yamlNode.Content[i+1]

			// Handle "<<" keys
			if key.Tag == "!!merge" {
				if err := result.merge(config, yamlNode, key, value, mergeExemptions); err != nil {
					return nil, err
				}

				continue
			}

			// Apparently this is possible, so do the sanity check
			if key.Tag != "!!str" {
				return nil, parsererror.NewRich(yamlNode.Line, yamlNode.Column, "map key is not a string")
			}

			mapSubtree, err := convert(config, result, key.Value, value, key.Line, key.Column, mergeExemptions)
			if err != nil {
				return nil, err
			}

			result.Children = append(result.Children, mapSubtree)
		}
	case yaml.ScalarNode:
		result.Value = &ScalarValue{Value: yamlNode.Value}
	case yaml.AliasNode:
		// test for recursion
		for aParent := parent; aParent != nil; aParent = aParent.Parent {
			// If we've found a parent with the same anchor, it means we have a recursion
			if aParent.YAMLNode != nil && aParent.YAMLNode.Anchor == yamlNode.Value {
				return nil, parsererror.NewRich(yamlNode.Line, yamlNode.Column, "recursive alias '%s'", yamlNode.Value)
			}
		}

		// YAML aliases generally don't need line and column helper values
		// since they are merged into some other data structure afterwards
		// and this helps to find bugs easier in the future
		resolvedAlias, err := convert(config, parent, name, yamlNode.Alias, 0, 0, mergeExemptions)
		if err != nil {
			return nil, err
		}

		return resolvedAlias, nil
	default:
		return nil, parsererror.NewRich(yamlNode.Line, yamlNode.Column, "unexpected %s",
			yamlKindToString(yamlNode.Kind))
	}

	return result, nil
}

func (node *Node) merge(
	config *config,
	parent *yaml.Node,
	_ *yaml.Node,
	value *yaml.Node,
	mergeExemptions []nameable.Nameable,
) error {
	// YAML aliases generally don't need line and column helper values
	// since they are merged into some other data structure afterwards
	// and this helps to find bugs easier in the future
	aliasValue, err := convert(config, node, "", value, 0, 0, mergeExemptions)
	if err != nil {
		return err
	}

	if value.Kind == yaml.AliasNode {
		// According to spec[1], a merge key "<<" can either be associated with a single mapping node
		// or a sequence.
		//
		// [1]: https://yaml.org/type/merge.html
		switch aliasValue.Value.(type) {
		case *MapValue:
			node.MergeFromMap(aliasValue, mergeExemptions)
		case *ListValue:
			node.OverwriteWith(aliasValue)
		default:
			return parsererror.NewRich(parent.Line, parent.Column,
				"merge key should either be associated with a mapping or a sequence")
		}
	} else if value.Kind == yaml.SequenceNode {
		// According to spec[1], if the value associated with the merge key is a sequence,
		// then this sequence is expected to contain mapping nodes and each of these nodes
		// is merged in turn according to its order in the sequence.
		//
		// [1]: https://yaml.org/type/merge.html
		for _, aliasValueChild := range aliasValue.Children {
			if !aliasValueChild.IsMap() {
				return parsererror.NewRich(parent.Line, parent.Column,
					"got a sequence as a merge key's value, but not all of it's entries are mappings"+
						" (as required per spec)")
			}

			node.MergeFromMap(aliasValueChild, mergeExemptions)
		}
	}

	return nil
}

func yamlKindToString(kind yaml.Kind) string {
	switch kind {
	case yaml.DocumentNode:
		return "document"
	case yaml.SequenceNode:
		return "sequence"
	case yaml.MappingNode:
		return "mapping"
	case yaml.ScalarNode:
		return "scalar"
	case yaml.AliasNode:
		return "alias"
	default:
		return fmt.Sprintf("element of kind %d", kind)
	}
}

func isEven(number int) bool {
	const divisor = 2

	return (number % divisor) == 0
}
