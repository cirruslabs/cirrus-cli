package node

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
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

func (node *Node) String() string {
	switch value := node.Value.(type) {
	case *MapValue:
		var children []string
		for _, child := range node.Children {
			children = append(children, fmt.Sprintf("%s=%s", child.Name, child.String()))
		}
		return fmt.Sprintf("MapValue(%s)", strings.Join(children, ", "))
	case *ListValue:
		var children []string
		for _, child := range node.Children {
			children = append(children, child.String())
		}
		return fmt.Sprintf("ListValue(%s)", strings.Join(children, ", "))
	case *ScalarValue:
		return fmt.Sprintf("ScalarValue(%s)", value.Value)
	default:
		return "UnknownNodeType()"
	}
}

func (node *Node) CopyWithParent(parent *Node) *Node {
	result := &Node{
		Name:   node.Name,
		Value:  node.Value,
		Parent: parent,
	}

	for _, child := range node.Children {
		result.Children = append(result.Children, child.CopyWithParent(result))
	}

	return result
}

func (node *Node) Deduplicate() {
	// Split children into two groups
	seen := map[string]*Node{}
	var unique, duplicate []*Node

	for _, child := range node.Children {
		if _, ok := seen[child.Name]; ok {
			duplicate = append(duplicate, child)
		} else {
			unique = append(unique, child)
			seen[child.Name] = child
		}
	}

	// Merge children from the duplicate group into their unique counterparts
	// with recursive descent
	for _, duplicateChild := range duplicate {
		duplicateChild.Deduplicate()

		uniqueChild := seen[duplicateChild.Name]
		uniqueChild.MergeFrom(duplicateChild)
	}

	node.Children = unique
}

func (node *Node) MergeListOfMapsToSingleMap() {
	_, ok := node.Value.(*ListValue)
	if !ok {
		return
	}

	var virtualNode Node

	for _, child := range node.Children {
		if _, ok := child.Value.(*MapValue); !ok {
			return
		}

		virtualNode.MergeFrom(child)
	}

	node.Children = virtualNode.Children

	// Rewrite parents from virtualNode to node
	for _, child := range virtualNode.Children {
		child.Parent = node
	}

	// This is now a map
	node.Value = &MapValue{}
}

func (node *Node) MergeFrom(other *Node) {
	node.Name = other.Name

	// Special treatment for environment variables since they can also be represented as a list of maps
	if node.Name == "env" || node.Name == "environment" {
		node.MergeListOfMapsToSingleMap()
		other.MergeListOfMapsToSingleMap()
	}

	if reflect.TypeOf(node.Value) != reflect.TypeOf(other.Value) {
		node.Value = other.Value

		// Simply overwrite node's children with other's children
		node.Children = node.Children[:0]
		for _, child := range other.Children {
			node.Children = append(node.Children, child.CopyWithParent(node))
		}
		return
	}

	switch other.Value.(type) {
	case *MapValue:
		existingChildren := map[string]*Node{}

		for _, child := range node.Children {
			existingChildren[child.Name] = child
		}

		for _, otherChild := range other.Children {
			existingChild, ok := existingChildren[otherChild.Name]
			if ok {
				existingChild.MergeFrom(otherChild)
			} else {
				node.Children = append(node.Children, otherChild.CopyWithParent(node))
			}
		}
	case *ListValue:
		for i, otherChild := range other.Children {
			if i < len(node.Children) {
				// We have a counterpart child, do a merge
				node.Children[i].MergeFrom(otherChild)
			} else {
				// They have more children that we do, simply append them one by one
				node.Children = append(node.Children, otherChild.CopyWithParent(node))
			}
		}
	case *ScalarValue:
		node.Value = other.Value
	}
}
