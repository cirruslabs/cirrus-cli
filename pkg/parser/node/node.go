package node

import (
	"fmt"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
	"reflect"
	"strings"
)

type Node struct {
	Name     string
	Value    interface{}
	Parent   *Node
	Children []*Node

	Line   int
	Column int

	YAMLNode *yaml.Node
}

type MapValue struct{}
type ListValue struct{}
type ScalarValue struct {
	Value string
}

func (node *Node) ValueTypeAsString() string {
	switch node.Value.(type) {
	case *MapValue:
		return "map"
	case *ListValue:
		return "list"
	case *ScalarValue:
		return "scalar"
	default:
		return "unknown"
	}
}

func (node *Node) ValueIsEmpty() bool {
	switch value := node.Value.(type) {
	case *MapValue, *ListValue:
		return len(node.Children) == 0
	case *ScalarValue:
		return value.Value == ""
	default:
		return false
	}
}

func (node *Node) IsMap() bool {
	_, isMap := node.Value.(*MapValue)

	return isMap
}

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
		Line:   node.Line,
		Column: node.Column,
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
	node.Line = other.Line
	node.Column = other.Column

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
		// The strange logic below is only needed to make the tests
		// overriding the "additional_containers:" field pass.
		//
		// However, it works very awkwardly for a lot of other
		// use-cases, because, well, merging lists like that is
		// just asking for it.
		//
		// Let's only allow this logic for "additional_containers:"
		// and deprecate for the rest of the fields.
		if node.Name == "additional_containers" {
			for i, otherChild := range other.Children {
				if i < len(node.Children) {
					// We have a counterpart child, do a merge
					node.Children[i].MergeFrom(otherChild)
				} else {
					// They have more children that we do, simply append them one by one
					node.Children = append(node.Children, otherChild.CopyWithParent(node))
				}
			}

			break
		}

		node.Children = lo.Map(other.Children, func(child *Node, _ int) *Node {
			return child.CopyWithParent(node)
		})
	case *ScalarValue:
		node.Value = other.Value
	}
}
