package larker

import (
	yamlhelpers "github.com/cirruslabs/cirrus-cli/pkg/helpers/yaml"
	"go.starlark.net/starlark"
	"gopkg.in/yaml.v3"
	"strings"
)

func convertTasks(starlarkTasks *starlark.List) *yaml.Node {
	yamlList := convertList(starlarkTasks)

	if yamlList == nil || len(yamlList.Content) == 0 {
		return nil
	}

	// Adapt a list of tasks to a YAML configuration format that expects a map on it's outer layer
	var serializableMainResult []*yaml.Node
	for _, listItem := range yamlList.Content {
		serializableMainResult = append(serializableMainResult, yamlhelpers.NewStringNode("task"))
		serializableMainResult = append(serializableMainResult, listItem)
	}

	return yamlhelpers.NewMapNode(serializableMainResult)
}

func convertList(l *starlark.List) *yaml.Node {
	iter := l.Iterate()
	defer iter.Done()

	var listValue starlark.Value

	var items []*yaml.Node
	for iter.Next(&listValue) {
		switch value := listValue.(type) {
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

	return yamlhelpers.NewSeqNode(items)
}

func convertDict(d *starlark.Dict) *yaml.Node {
	var items []*yaml.Node

	for _, dictTuple := range d.Items() {
		items = append(items, yamlhelpers.NewStringNode(strings.Trim(dictTuple[0].String(), "'\"")))

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

	return yamlhelpers.NewMapNode(items)
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
