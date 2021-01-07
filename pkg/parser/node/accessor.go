package node

import (
	"encoding/base64"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"golang.org/x/text/encoding/unicode"
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

func (node *Node) GetBoolValue(env map[string]string, boolevator *boolevator.Boolevator) (bool, error) {
	expression, err := node.GetStringValue()
	if err != nil {
		return false, err
	}

	evaluation, err := boolevator.Eval(expression, env)
	if err != nil {
		return false, err
	}

	return evaluation, nil
}

func (node *Node) GetExpandedStringValue(env map[string]string) (string, error) {
	valueNode, ok := node.Value.(*ScalarValue)
	if !ok {
		return "", fmt.Errorf("%w: not a scalar value", parsererror.ErrParsing)
	}

	return ExpandEnvironmentVariables(valueNode.Value, env), nil
}

func (node *Node) GetSliceOfStrings() ([]string, error) {
	_, ok := node.Value.(*ListValue)
	if !ok {
		return nil, fmt.Errorf("%w: expected %s node to be a list", parsererror.ErrParsing, node.Name)
	}

	var result []string

	for _, child := range node.Children {
		scalar, ok := child.Value.(*ScalarValue)
		if !ok {
			return nil, fmt.Errorf("%w: %s node's list items should be scalars", parsererror.ErrParsing, node.Name)
		}

		result = append(result, scalar.Value)
	}

	return result, nil
}

func (node *Node) GetSliceOfExpandedStrings(env map[string]string) ([]string, error) {
	sliceStrings, err := node.GetSliceOfNonEmptyStrings()
	if err != nil {
		return nil, err
	}

	var result []string

	for _, sliceString := range sliceStrings {
		result = append(result, ExpandEnvironmentVariables(sliceString, env))
	}

	return result, nil
}

func (node *Node) GetExpandedStringMapping(env map[string]string) (map[string]string, error) {
	mapping, err := node.GetStringMapping()
	if err != nil {
		return mapping, err
	}
	result := make(map[string]string, len(mapping))
	for key, value := range mapping {
		result[key] = ExpandEnvironmentVariables(value, env)
	}
	return result, nil
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
		return node.GetSliceOfStrings()
	default:
		return nil, fmt.Errorf("%w: field should be a string or a list of values", parsererror.ErrParsing)
	}
}

func (node *Node) GetScript() ([]string, error) {
	switch value := node.Value.(type) {
	case *ScalarValue:
		return strings.Split(value.Value, "\n"), nil
	case *ListValue:
		var result []string

		for _, child := range node.Children {
			switch childValue := child.Value.(type) {
			case *ScalarValue:
				result = append(result, strings.Split(childValue.Value, "\n")...)
			case *MapValue:
				// support powershell trick
				psValueNode := child.FindChild("ps")
				if psValueNode == nil {
					return nil, fmt.Errorf("%w: script only supports 'ps: ' helper syntax for Powershell", parsererror.ErrParsing)
				}
				psValue, err := psValueNode.GetStringValue()
				if err != nil {
					return nil, fmt.Errorf("%w: failed to get Powershell script (%v)", parsererror.ErrParsing, err)
				}
				encoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
				valueBytes, err := encoder.Bytes([]byte(psValue))
				if err != nil {
					return nil, fmt.Errorf("%w: failed to encode Powershell script (%v)", parsererror.ErrParsing, err)
				}
				encodedValue := base64.StdEncoding.EncodeToString(valueBytes)
				result = append(result, fmt.Sprintf("powershell.exe -NoLogo -EncodedCommand %s", encodedValue))
			}
		}

		return result, nil
	default:
		return nil, fmt.Errorf("%w: field should be a string or a list of values", parsererror.ErrParsing)
	}
}

func (node *Node) GetEnvironment() (map[string]string, error) {
	switch node.Value.(type) {
	case *ListValue:
		accumulatedEnv := make(map[string]string)

		for _, child := range node.Children {
			childEnv, err := child.GetStringMapping()
			if err != nil {
				return nil, err
			}

			accumulatedEnv = environment.Merge(accumulatedEnv, childEnv)
		}

		return accumulatedEnv, nil
	case *MapValue:
		return node.GetStringMapping()
	default:
		return nil, fmt.Errorf("%w: field should be a map or a list of maps", parsererror.ErrParsing)
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
