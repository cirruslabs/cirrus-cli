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
	var key string
	var item *yaml.Node

	for iter.Next(&listValue) {
		pair, ok := listValue.(starlark.Tuple)
		if ok && pair.Len() == 2 {
			// Cirrus accepts repeated keys in a YAML Mapping, but that is not
			// allowed in Starlark. The closest we can have is a list of tuples: [(key, value)]
			keyObject := pair.Index(0).(starlark.String)
			key = keyObject.GoString()
			item = convertValue(pair.Index(1))
		} else {
			key = "task"
			item = convertValue(listValue)
		}
		serializableMainResult = append(serializableMainResult, yamlhelper.NewStringNode(key))
		serializableMainResult = append(serializableMainResult, item)
	}

	return yamlhelper.NewMapNode(serializableMainResult)
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
