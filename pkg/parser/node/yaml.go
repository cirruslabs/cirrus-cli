package node

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"strings"
)

var ErrFailedToMarshal = errors.New("failed to marshal to YAML")

const DefaultYamlMarshalIndent = 2

func NewFromText(text string) (*Node, error) {
	var yamlNode yaml.Node

	// Unmarshal YAML
	if err := yaml.Unmarshal([]byte(text), &yamlNode); err != nil {
		return nil, err
	}

	// Convert the parsed and nested YAML structure into a tree
	// to get the ability to walk parents

	if yamlNode.Kind == 0 || len(yamlNode.Content) == 0 {
		// Empty document
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

func NewNodeFromScalar(value interface{}) *Node {
	return &Node{
		Value: &ScalarValue{
			Value: value,
		},
	}
}

func NewNodeMap(content []*Node) *Node {
	return &Node{
		Value:    &MapValue{},
		Children: content,
	}
}

func NewNodeList(content []*Node) *Node {
	return &Node{
		Value:    &ListValue{},
		Children: content,
	}
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

func (node *Node) MarshalPrettyYAML() (string, error) {
	builder := &strings.Builder{}
	encoder := yaml.NewEncoder(builder)
	encoder.SetIndent(DefaultYamlMarshalIndent)
	nodes, err := node.marshalYAML()
	if err != nil {
		return "", fmt.Errorf("%w: cannot convert into YAML: %v", ErrFailedToMarshal, err)
	}
	err = encoder.Encode(newMapNode(nodes))
	if err != nil {
		return "", fmt.Errorf("%w: cannot marshal into YAML: %v", ErrFailedToMarshal, err)
	}
	err = encoder.Close()
	if err != nil {
		return "", fmt.Errorf("%w: cannot finish marshaling into YAML: %v", ErrFailedToMarshal, err)
	}

	return builder.String(), nil
}

func (node *Node) marshalYAML() ([]*yaml.Node, error) {
	switch obj := node.Value.(type) {
	case *MapValue:
		var resultChildren []*yaml.Node
		for _, child := range node.Children {
			marshalledItem, err := child.marshalYAML()
			if err != nil {
				return nil, err
			}

			resultChildren = append(resultChildren, marshalledItem...)
		}

		return []*yaml.Node{newStringNode(node.Name), newMapNode(resultChildren)}, nil
	case *ListValue:
		var resultChildren []*yaml.Node
		for _, child := range node.Children {
			marshalledItem, err := child.marshalYAML()
			if err != nil {
				return nil, err
			}

			resultChildren = append(resultChildren, marshalledItem...)
		}

		return []*yaml.Node{newStringNode(node.Name), newSeqNode(resultChildren)}, nil
	case *ScalarValue:
		if node.Name != "" {
			return []*yaml.Node{newStringNode(node.Name), newScalarNode(obj.Value)}, nil
		} else {
			return []*yaml.Node{newScalarNode(obj.Value)}, nil
		}
	default:
		return nil, fmt.Errorf("%w: unknown node type: %T", ErrFailedToMarshal, node.Value)
	}
}

func newSeqNode(content []*yaml.Node) *yaml.Node {
	var result yaml.Node
	result.Kind = yaml.SequenceNode
	result.Tag = "!!seq"
	result.Content = content
	return &result
}

func newMapNode(content []*yaml.Node) *yaml.Node {
	var result yaml.Node
	result.Kind = yaml.MappingNode
	result.Tag = "!!map"
	result.Content = content
	return &result
}

func newStringNode(text string) *yaml.Node {
	var result yaml.Node
	result.SetString(text)
	return &result
}

func newScalarNode(value interface{}) *yaml.Node {
	var result yaml.Node
	result.Kind = yaml.ScalarNode
	result.Value = fmt.Sprintf("%v", value)
	result.Tag = "!!str"
	switch value.(type) {
	case int:
		result.Tag = "!!int"
	case float64:
		result.Tag = "!!float"
	case bool:
		result.Tag = "!!bool"
	}
	return &result
}
