package node

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"os"
)

func (node *Node) GetStringValue() (string, error) {
	valueNode, ok := node.Value.(*ScalarValue)
	if !ok {
		return "", fmt.Errorf("%w: not a scalar value", parsererror.ErrParsing)
	}

	return valueNode.Value, nil
}

func (node *Node) GetExpandedStringValue(env map[string]string) (string, error) {
	valueNode, ok := node.Value.(*ScalarValue)
	if !ok {
		return "", fmt.Errorf("%w: not a scalar value", parsererror.ErrParsing)
	}

	return os.Expand(valueNode.Value, func(key string) string {
		result, ok := env[key]
		if !ok {
			return ""
		}
		return result
	}), nil
}

func (node *Node) GetStringMapping() (map[string]string, error) {
	result := make(map[string]string)

	if _, ok := node.Value.(*MapValue); !ok {
		return nil, fmt.Errorf("%w: attempted to retrieve mapping for a non-mapping node", parsererror.ErrParsing)
	}

	for _, child := range node.Children {
		scalarValue, ok := child.Value.(*ScalarValue)
		if !ok {
			return nil, fmt.Errorf("%w: attempted to retrieve mapping for a mapping node with non-scalar values",
				parsererror.ErrParsing)
		}

		result[child.Name] = scalarValue.Value
	}

	return result, nil
}

func (node *Node) GetSliceOfNonEmptyStrings() ([]string, error) {
	switch value := node.Value.(type) {
	case *ScalarValue:
		return []string{value.Value}, nil
	case *ListValue:
		var result []string
		for _, child := range node.Children {
			value2, ok := child.Value.(*ScalarValue)
			if !ok {
				return nil, fmt.Errorf("%w: list items should be scalars", parsererror.ErrParsing)
			}
			result = append(result, value2.Value)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("%w: field should be a string or a list of values", parsererror.ErrParsing)
	}
}
