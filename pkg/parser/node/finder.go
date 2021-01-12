package node

func (node *Node) DeepFindCollectible(name string) *Node {
	var fulfilledAtLeastOnce bool
	var virtualNode Node

	// Accumulate collectible nodes in descending order of priority
	// (e.g. "env" field from the root YAML map comes first, then "env" field from the task's map, etc.)
	var traverseChain []*Node

	for current := node; current != nil; current = current.Parent {
		for i := len(current.Children) - 1; i >= 0; i-- {
			child := current.Children[i]

			if child.Name == name {
				traverseChain = append(traverseChain, child)
			}
		}
	}

	// Starting from the lowest priority node,
	// merge the rest of the nodes into it
	for i := len(traverseChain) - 1; i >= 0; i-- {
		child := traverseChain[i]

		virtualNode.MergeFrom(child)

		if !fulfilledAtLeastOnce {
			fulfilledAtLeastOnce = true
		}
	}

	if !fulfilledAtLeastOnce {
		return nil
	}

	// Simulate Cirrus Cloud parser behavior
	virtualNode.Deduplicate()
	// link to the tree so collectible sub-fields will work
	virtualNode.Parent = node

	return &virtualNode
}

func (node *Node) HasChild(name string) bool {
	return node.FindChild(name) != nil
}

func (node *Node) FindChild(name string) *Node {
	for _, child := range node.Children {
		if child.Name == name {
			return child
		}
	}

	return nil
}
