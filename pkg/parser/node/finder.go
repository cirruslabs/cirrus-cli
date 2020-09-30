package node

func (node *Node) DeepFindChild(name string) *Node {
	var fulfilledAtLeastOnce bool
	var virtualNode Node

	for current := node; current != nil; current = current.Parent {
		for i := len(current.Children) - 1; i >= 0; i-- {
			child := current.Children[i]

			if child.Name != name {
				continue
			}

			if !fulfilledAtLeastOnce {
				virtualNode = *child
				fulfilledAtLeastOnce = true
			}

			for i := len(child.Children) - 1; i >= 0; i-- {
				subChild := child.Children[i]

				// Append fields from child that we don't have
				if !virtualNode.HasChild(subChild.Name) {
					virtualNode.Children = append(virtualNode.Children, subChild)
				}
			}

			break
		}
	}

	if !fulfilledAtLeastOnce {
		return nil
	}

	return &virtualNode
}

func (node *Node) HasChild(name string) bool {
	for _, child := range node.Children {
		if child.Name == name {
			return true
		}
	}

	return false
}
