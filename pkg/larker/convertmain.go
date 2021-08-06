package larker

import (
	"github.com/cirruslabs/cirrus-cli/pkg/yamlhelper"
	"go.starlark.net/starlark"
	"gopkg.in/yaml.v3"
	"strings"
)

func convertInstructions(instructions *starlark.List) *yaml.Node {
	if instructions == nil || instructions.Len() == 0 {
		return nil
	}

	iter := instructions.Iterate()
	defer iter.Done()

	var listValue starlark.Value
	var serializableMainResult []*yaml.Node

	for iter.Next(&listValue) {
		k, v := listValueToKV(listValue)
		serializableMainResult = append(serializableMainResult, yamlhelper.NewStringNode(k))
		serializableMainResult = append(serializableMainResult, v)
	}

	return yamlhelper.NewMapNode(serializableMainResult)
}

func listValueToKV(listValue starlark.Value) (string, *yaml.Node) {
	// YAML configuration freely accepts duplicate map keys, but that is not allowed in Starlark,
	// so we support an alternative syntax where the map key and value are decomposed into a tuple:
	// [(key, value)]
	// ...which additionally allows us to express anything and not just tasks.
	tuple, ok := listValue.(starlark.Tuple)
	if ok && tuple.Len() == 2 {
		key, ok := tuple.Index(0).(starlark.String)
		if ok {
			return key.GoString(), convertValue(tuple.Index(1))
		}
	}

	return "task", convertValue(listValue)
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
