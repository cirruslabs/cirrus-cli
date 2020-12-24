package node

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
)

type Node struct {
	Name     string
	Value    interface{}
	Parent   *Node
	Children []*Node
}

type MapValue struct{}
type ListValue struct{}
type ScalarValue struct {
	Value string
}

var ErrNodeConversionFailed = errors.New("node conversion failed")

func NewFromSlice(slice yaml.MapSlice) (*Node, error) {
	return convert(nil, "root", slice)
}

func convert(parent *Node, name string, obj interface{}) (*Node, error) {
	var result *Node

	switch typedObj := obj.(type) {
	case []interface{}:
		result = &Node{
			Name:   name,
			Value:  &ListValue{},
			Parent: parent,
		}

		for _, arrayItem := range typedObj {
			listSubtree, err := convert(result, "", arrayItem)
			if err != nil {
				return nil, err
			}

			result.Children = append(result.Children, listSubtree)
		}
	case yaml.MapSlice:
		result = &Node{
			Name:   name,
			Value:  &MapValue{},
			Parent: parent,
		}

		for _, sliceItem := range typedObj {
			// Apparently this is possible, so do the sanity check
			key, ok := sliceItem.Key.(string)
			if !ok {
				return nil, fmt.Errorf("%w: map key is not a string", ErrNodeConversionFailed)
			}

			mapSubtree, err := convert(result, key, sliceItem.Value)
			if err != nil {
				return nil, err
			}

			result.Children = append(result.Children, mapSubtree)
		}
	default:
		scalar := &ScalarValue{}
		if typedObj != nil {
			scalar.Value = fmt.Sprintf("%v", typedObj)
		}
		result = &Node{
			Name:   name,
			Value:  scalar,
			Parent: parent,
		}
	}

	return result, nil
}
