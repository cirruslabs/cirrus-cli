package larker

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/utils"
	"go.starlark.net/starlark"
	"gopkg.in/yaml.v3"
	"strings"
)

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

	return utils.NewSeqNode(items)
}

func convertDict(d *starlark.Dict) *yaml.Node {
	var items []*yaml.Node

	for _, dictTuple := range d.Items() {
		items = append(items, utils.NewStringNode(strings.Trim(dictTuple[0].String(), "'\"")))

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

	return utils.NewMapNode(items)
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
