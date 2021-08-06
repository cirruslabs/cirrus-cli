package larker

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/yamlhelper"
	"go.starlark.net/starlark"
	"gopkg.in/yaml.v3"
	"strings"
)

var (
	ErrMalformedKeyValueTuple = errors.New("malformed key-value tuple")
	ErrNotADictOrTuple        = errors.New("main() should return a list of either dicts or tuples")
)

func convertInstructions(instructions *starlark.List) (*yaml.Node, error) {
	if instructions == nil || instructions.Len() == 0 {
		return nil, nil
	}

	iter := instructions.Iterate()
	defer iter.Done()

	var listValue starlark.Value
	var serializableMainResult []*yaml.Node

	for iter.Next(&listValue) {
		k, v, err := listValueToKV(listValue)
		if err != nil {
			return nil, err
		}
		serializableMainResult = append(serializableMainResult, yamlhelper.NewStringNode(k))
		serializableMainResult = append(serializableMainResult, v)
	}

	return yamlhelper.NewMapNode(serializableMainResult), nil
}

func listValueToKV(listValue starlark.Value) (string, *yaml.Node, error) {
	switch value := listValue.(type) {
	case starlark.Tuple:
		// YAML configuration freely accepts duplicate map keys, but that is not allowed in Starlark,
		// so we support an alternative syntax where the map key and value are decomposed into a tuple:
		// [(key, value)]
		// ...which additionally allows us to express anything and not just tasks.

		if value.Len() != 2 {
			return "", nil, fmt.Errorf("%w: tuple should contain exactly 2 elements",
				ErrMalformedKeyValueTuple)
		}

		key, ok := value.Index(0).(starlark.String)
		if !ok {
			return "", nil, fmt.Errorf("%w: first tuple element should be a string",
				ErrMalformedKeyValueTuple)
		}

		return key.GoString(), convertValue(value.Index(1)), nil
	case *starlark.Dict:
		return "task", convertValue(value), nil
	default:
		return "", nil, ErrNotADictOrTuple
	}
}

func convertValue(v starlark.Value) *yaml.Node {
	switch value := v.(type) {
	case *starlark.List:
		return convertList(value)
	case *starlark.Dict:
		return convertDict(value)
	default:
		var valueNode yaml.Node
		_ = valueNode.Encode(convertPrimitive(value))
		return &valueNode
	}
}

func convertList(l *starlark.List) *yaml.Node {
	iter := l.Iterate()
	defer iter.Done()

	var listValue starlark.Value

	var items []*yaml.Node
	for iter.Next(&listValue) {
		items = append(items, convertValue(listValue))
	}

	return yamlhelper.NewSeqNode(items)
}

func convertDict(d *starlark.Dict) *yaml.Node {
	var items []*yaml.Node

	for _, dictTuple := range d.Items() {
		items = append(items, yamlhelper.NewStringNode(strings.Trim(dictTuple[0].String(), "'\"")))

		switch value := dictTuple[1].(type) {
		case *starlark.List:
			items = append(items, convertList(value))
		case *starlark.Dict:
			items = append(items, convertDict(value))
		default:
			var valueNode yaml.Node
			_ = valueNode.Encode(convertPrimitive(value))
			items = append(items, &valueNode)
		}
	}

	return yamlhelper.NewMapNode(items)
}

func convertPrimitive(value starlark.Value) interface{} {
	switch typedValue := value.(type) {
	case starlark.Int:
		res, _ := typedValue.Int64()
		return res
	default:
		return typedValue
	}
}
