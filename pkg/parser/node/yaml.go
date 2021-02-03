package node

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/yamlhelper"
	"gopkg.in/yaml.v3"
)

var ErrFailedToMarshal = errors.New("failed to marshal to YAML")

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

func NewFromText(text string) (*Node, error) {
	var yamlNode yaml.Node

	// Unmarshal YAML
	if err := yaml.Unmarshal([]byte(text), &yamlNode); err != nil {
		return nil, err
	}

	// Empty document
	if yamlNode.Kind == 0 || len(yamlNode.Content) == 0 {
		return &Node{Name: "root"}, nil
	}

	if yamlNode.Kind != yaml.DocumentNode {
		return nil, fmt.Errorf("%w: expected a YAML document, but got %s at %d:%d", ErrNodeConversionFailed,
			yamlKindToString(yamlNode.Kind), yamlNode.Line, yamlNode.Column)
	}

	if len(yamlNode.Content) > 1 {
		extraneous := yamlNode.Content[1]

		return nil, fmt.Errorf("%w: YAML document contains extraneous top-level elements,"+
			" such as %s at %d:%d", ErrNodeConversionFailed, yamlKindToString(extraneous.Kind),
			extraneous.Line, extraneous.Column)
	}

	if yamlNode.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%w: YAML document should contain a mapping as it top-level element,"+
			" but found %s at %d:%d", ErrNodeConversionFailed, yamlKindToString(yamlNode.Kind),
			yamlNode.Line, yamlNode.Column)
	}

	return convert(nil, "root", yamlNode.Content[0])
}

// nolint:gocognit // splitting this into multiple functions would probably make this even less comprehensible
func convert(parent *Node, name string, yamlNode *yaml.Node) (*Node, error) {
	result := &Node{
		Name:   name,
		Parent: parent,
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

			listSubtree, err := convert(result, "", item)
			if err != nil {
				return nil, err
			}

			result.Children = append(result.Children, listSubtree)
		}
	case yaml.MappingNode:
		result.Value = &MapValue{}

		if !isEven(len(yamlNode.Content)) {
			return nil, fmt.Errorf("%w: unbalanced map at %d:%d", ErrNodeConversionFailed,
				yamlNode.Line, yamlNode.Column)
		}

		for i := 0; i < len(yamlNode.Content); i += 2 {
			// Handle "<<" keys
			if yamlNode.Content[i].Tag == "!!merge" {
				aliasValue, err := convert(result, "", yamlNode.Content[i+1])
				if err != nil {
					return nil, err
				}

				result.MergeMapsOrOverwrite(aliasValue)

				continue
			}

			// Apparently this is possible, so do the sanity check
			if yamlNode.Content[i].Tag != "!!str" {
				return nil, fmt.Errorf("%w: map key is not a string", ErrNodeConversionFailed)
			}

			mapSubtree, err := convert(result, yamlNode.Content[i].Value, yamlNode.Content[i+1])
			if err != nil {
				return nil, err
			}

			result.Children = append(result.Children, mapSubtree)
		}
	case yaml.ScalarNode:
		result.Value = &ScalarValue{Value: yamlNode.Value}
	case yaml.AliasNode:
		resolvedAlias, err := convert(parent, name, yamlNode.Alias)
		if err != nil {
			return nil, err
		}

		return resolvedAlias, nil
	default:
		return nil, fmt.Errorf("%w: unexpected %s at %d:%d", ErrNodeConversionFailed,
			yamlKindToString(yamlNode.Kind), yamlNode.Line, yamlNode.Column)
	}

	return result, nil
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
