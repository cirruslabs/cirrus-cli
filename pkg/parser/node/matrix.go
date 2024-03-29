package node

import "github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"

type Predicate func(nodeName string) bool

func (node *Node) FindParent(predicate Predicate) *Node {
	current := node.Parent

	for current != nil {
		if predicate(current.Name) {
			return current
		}

		current = current.Parent
	}

	return nil
}

func ReversePath(path []int) []int {
	var result []int

	if len(path) == 0 {
		return []int{}
	}

	for i := len(path) - 1; i >= 0; i-- {
		result = append(result, path[i])
	}

	return result
}

func (node *Node) PathUpwardsUpto(upto *Node) []int {
	var result []int

	parent := node.Parent
	lookingFor := node

	for parent != nil {
		for i, parentChild := range parent.Children {
			if parentChild == lookingFor {
				result = append(result, i)
				lookingFor = parent
				break
			}
		}

		if parent == upto {
			break
		}

		parent = parent.Parent
	}

	return result
}

func (node *Node) GetPath(path []int) *Node {
	found := node

	for _, index := range path {
		if len(found.Children) < index+1 {
			return nil
		}

		found = found.Children[index]
	}

	return found
}

func (node *Node) DeepCopy() *Node {
	return node.CopyWithParent(nil)
}

func (node *Node) DeepCopyWithReplacements(target *Node, replacements []*Node) *Node {
	// Remember the path to the target node so we'll be able
	// to find it in the deep copy
	pathToNode := ReversePath(target.PathUpwardsUpto(node))

	nodeCopy := node.DeepCopy()
	targetCopy := nodeCopy.GetPath(pathToNode)

	targetCopy.ReplaceWith(replacements)

	return nodeCopy
}

func (node *Node) ReplaceWith(with []*Node) {
	var newChildren []*Node

	// Link replacements to the node's parent
	for _, withItem := range with {
		withItem.Parent = node.Parent
	}

	for _, thisLevelChild := range node.Parent.Children {
		if thisLevelChild == node {
			newChildren = append(newChildren, with...)
		} else {
			newChildren = append(newChildren, thisLevelChild)
		}
	}

	node.Parent.Children = newChildren
}

func (node *Node) MergeFromMap(with *Node, mergeExemptions []nameable.Nameable) {
	// If the value associated with the key is a single mapping node,
	// each of its key/value pairs is inserted into the current mapping,
	// unless the key already exists in it[1] OR the key is associated with
	// a collectible or repeatable field.
	//
	// [1]: https://yaml.org/type/merge.html
	for _, child := range with.Children {
		var mergeExemption bool

		for _, nameable := range mergeExemptions {
			if nameable.Matches(child.Name) {
				mergeExemption = true
				break
			}
		}

		if node.FindChild(child.Name) == nil || mergeExemption {
			child.Parent = node
			node.Children = append(node.Children, child)
		}
	}
}

func (node *Node) OverwriteWith(with *Node) {
	node.Value = with.Value
	node.Children = node.Children[:0]
	for _, child := range with.Children {
		child.Parent = node
		node.Children = append(node.Children, child)
	}
}
