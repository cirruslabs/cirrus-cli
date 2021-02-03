package larker

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"go.starlark.net/starlark"
	"strings"
)

func convertList(l *starlark.List) *node.Node {
	iter := l.Iterate()
	defer iter.Done()
	return convertIterator(iter)
}

func convertIterator(iter starlark.Iterator) *node.Node {
	var listValue starlark.Value

	var items []*node.Node
	for iter.Next(&listValue) {
		switch value := listValue.(type) {
		case *starlark.List:
			items = append(items, convertList(value))
		case starlark.Tuple:
			items = append(items, convertIterator(value.Iterate()))
		case *starlark.Dict:
			items = append(items, convertDict(value))
		default:
			items = append(items, node.NewNodeFromScalar(convertPrimitive(value)))
		}
	}

	return node.NewNodeList(items)
}

func convertDict(d *starlark.Dict) *node.Node {
	var items []*node.Node

	for _, dictTuple := range d.Items() {
		var currentNode *node.Node

		switch value := dictTuple[1].(type) {
		case *starlark.List:
			currentNode = convertList(value)
		case starlark.Tuple:
			currentNode = convertIterator(value.Iterate())
		case *starlark.Dict:
			currentNode = convertDict(value)
		default:
			currentNode = node.NewNodeFromScalar(convertPrimitive(value))
		}
		currentNode.Name = strings.Trim(dictTuple[0].String(), "'\"")
		items = append(items, currentNode)
	}

	return node.NewNodeMap(items)
}

func convertPrimitive(value starlark.Value) interface{} {
	switch typedValue := value.(type) {
	case starlark.Int:
		res, _ := typedValue.Int64()
		return res
	case starlark.Float:
		return float64(typedValue)
	case starlark.Bool:
		return bool(typedValue)
	case starlark.String:
		return typedValue.GoString()
	default:
		return typedValue
	}
}
