package larker

import (
	"go.starlark.net/starlark"
	"gopkg.in/yaml.v2"
)

func convertList(l *starlark.List) (result []interface{}) {
	iter := l.Iterate()
	defer iter.Done()

	var listValue starlark.Value

	for iter.Next(&listValue) {
		switch value := listValue.(type) {
		case *starlark.List:
			result = append(result, convertList(value))
		case *starlark.Dict:
			result = append(result, convertDict(value))
		default:
			result = append(result, value)
		}
	}

	return
}

func convertDict(d *starlark.Dict) yaml.MapSlice {
	var slice yaml.MapSlice

	for _, dictTuple := range d.Items() {
		var sliceItem yaml.MapItem

		key := dictTuple[0]

		switch value := dictTuple[1].(type) {
		case *starlark.List:
			sliceItem = yaml.MapItem{Key: key, Value: convertList(value)}
		case *starlark.Dict:
			sliceItem = yaml.MapItem{Key: key, Value: convertDict(value)}
		default:
			sliceItem = yaml.MapItem{Key: key, Value: value}
		}

		slice = append(slice, sliceItem)
	}

	return slice
}
