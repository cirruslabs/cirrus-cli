package node

func (node *Node) DeepFindChildren(name string) *Node {
	var fulfilledAtLeastOnce bool
	var virtualNode Node

	for current := node; current != nil; current = current.Parent {
		for _, child := range current.Children {
			if child.Name == name {
				if !fulfilledAtLeastOnce {
					virtualNode = *child
					fulfilledAtLeastOnce = true
				}

				for _, subChild := range child.Children {
					// Append fields from child that we don't have
					if !virtualNode.HasChildren(subChild.Name) {
						virtualNode.Children = append(virtualNode.Children, subChild)
					}
				}
				break
			}
		}
	}

	if !fulfilledAtLeastOnce {
		return nil
	}

	return &virtualNode
}

func (node *Node) HasChildren(name string) bool {
	for _, child := range node.Children {
		if child.Name == name {
			return true
		}
	}

	return false
}
