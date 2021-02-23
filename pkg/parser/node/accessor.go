package node

import (
	"encoding/base64"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/expander"
	"golang.org/x/text/encoding/unicode"
	"strings"
)

func (node *Node) GetStringValue() (string, error) {
	valueNode, ok := node.Value.(*ScalarValue)
	if !ok {
		return "", node.ParserError("not a scalar value")
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
		return "", node.ParserError("not a scalar value")
	}

	return expander.ExpandEnvironmentVariables(valueNode.Value, env), nil
}

func (node *Node) GetSliceOfStrings() ([]string, error) {
	_, ok := node.Value.(*ListValue)
	if !ok {
		return nil, node.ParserError("expected a list")
	}

	var result []string

	for _, child := range node.Children {
		scalar, ok := child.Value.(*ScalarValue)
		if !ok {
			return nil, node.ParserError("list items should be scalars")
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
		result = append(result, expander.ExpandEnvironmentVariables(sliceString, env))
	}

	return result, nil
}

func (node *Node) GetStringMapping() (map[string]string, error) {
	result := make(map[string]string)

	if _, ok := node.Value.(*MapValue); !ok {
		return nil, node.ParserError("expected a map")
	}

	for _, child := range node.Children {
		flattenedValue, err := child.FlattenedValue()
		if err != nil {
			return nil, err
		}

		result[child.Name] = flattenedValue
	}

	return result, nil
}

func (node *Node) FlattenedValue() (string, error) {
	switch obj := node.Value.(type) {
	case *ScalarValue:
		return obj.Value, nil
	case *ListValue:
		var listValues []string

		for _, child := range node.Children {
			scalar, ok := child.Value.(*ScalarValue)
			if !ok {
				return "", child.ParserError("list should only contain scalar values")
			}

			listValues = append(listValues, scalar.Value)
		}

		return strings.Join(listValues, "\n"), nil
	default:
		return "", node.ParserError("expected a scalar value or a list with scalar values")
	}
}

func (node *Node) GetSliceOfNonEmptyStrings() ([]string, error) {
	switch value := node.Value.(type) {
	case *ScalarValue:
		return []string{value.Value}, nil
	case *ListValue:
		return node.GetSliceOfStrings()
	default:
		return nil, node.ParserError("expected a scalar value or a list with scalar values")
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
				if psValueNode := child.FindChild("ps"); psValueNode != nil {
					// Support PowerShell ("ps: ") syntax
					psValue, err := psValueNode.GetStringValue()
					if err != nil {
						return nil, err
					}

					encoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()

					valueBytes, err := encoder.Bytes([]byte(psValue))
					if err != nil {
						return nil, child.ParserError("failed to encode Powershell script: %v", err)
					}

					encodedValue := base64.StdEncoding.EncodeToString(valueBytes)

					result = append(result, fmt.Sprintf("powershell.exe -NoLogo -EncodedCommand %s", encodedValue))
				} else {
					// Minimally support incorrect, but historically accepted syntax, when a list value is unquoted
					// and is treated as map, but then converted to a scalar by the parser silently, e.g.:
					//
					// script:
					//   - echo "TODO: fix   this"
					//
					// ...which is gets processed similar to:
					//
					// script:
					//   - "echo \"TODO: fix this\""
					//
					// Note that the spaces surrounding the ':' character get lost as a result of this conversion.
					for _, subChild := range child.Children {
						scalarValue, ok := subChild.Value.(*ScalarValue)
						if !ok {
							return nil, subChild.ParserError("unsupported script syntax")
						}

						result = append(result, fmt.Sprintf("%s: %s", subChild.Name, scalarValue.Value))
					}
				}
			}
		}

		return result, nil
	default:
		return nil, node.ParserError("expected a scalar value or a list with scalar values")
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
		return nil, node.ParserError("expected a map or a list of maps")
	}
}
