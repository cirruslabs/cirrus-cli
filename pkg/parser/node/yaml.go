package node

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
)

var ErrFailedToMarshal = errors.New("failed to marshal to YAML")

func (node *Node) MarshalYAML() (*yaml.Node, error) {
	switch obj := node.Value.(type) {
	case *MapValue:
		var result yaml.Node
		result.Kind = yaml.MappingNode
		result.Tag = "!!map"

		var resultChildren []*yaml.Node
		for _, child := range node.Children {
			marshalledItem, err := child.MarshalYAML()
			if err != nil {
				return nil, err
			}

			var keyNode yaml.Node
			keyNode.SetString(child.Name)

			resultChildren = append(resultChildren, &keyNode)
			resultChildren = append(resultChildren, marshalledItem)
		}

		result.Content = resultChildren
		return &result, nil
	case *ListValue:
		var result yaml.Node
		result.Kind = yaml.SequenceNode
		result.Tag = "!!seq"

		var resultChildren []*yaml.Node
		for _, child := range node.Children {
			marshalledItem, err := child.MarshalYAML()
			if err != nil {
				return nil, err
			}

			resultChildren = append(resultChildren, marshalledItem)
		}

		return &result, nil
	case *ScalarValue:
		var valueNode yaml.Node
		valueNode.SetString(obj.Value)
		return &valueNode, nil
	default:
		return nil, fmt.Errorf("%w: unknown node type: %T", ErrFailedToMarshal, node.Value)
	}
}
