package node

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"sort"
	"strings"
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

	return ExpandEnvironmentVariables(valueNode.Value, env), nil
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

func ExpandEnvironmentVariables(s string, env map[string]string) string {
	const maxExpansionIterations = 10

	// Create a sorted view of the environment map keys to expand the longest keys first
	// and avoid cases where "$CIRRUS_BRANCH" is expanded into "true_BRANCH", assuming
	// env = map[string]string{"CIRRUS": "true", "CIRRUS_BRANCH": "main"})
	var sortedKeys []string
	for key := range env {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return !(sortedKeys[i] < sortedKeys[j])
	})

	for i := 0; i < maxExpansionIterations; i++ {
		beforeExpansion := s

		for _, key := range sortedKeys {
			s = strings.ReplaceAll(s, fmt.Sprintf("$%s", key), env[key])
			s = strings.ReplaceAll(s, fmt.Sprintf("${%s}", key), env[key])
			s = strings.ReplaceAll(s, fmt.Sprintf("%%%s%%", key), env[key])
		}

		// Don't wait till the end of the loop if we are not progressing
		if s == beforeExpansion {
			break
		}
	}

	return s
}
