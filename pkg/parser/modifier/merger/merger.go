package merger

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
)

func merge(tree *node.Node) {
	switch tree.Value.(type) {
	case *node.MapValue:
		indexesToOmit := map[int]struct{}{}
		prevNameToIndex := map[string]int{}

		for i, child := range tree.Children {
			merge(child)

			prevIndex, hasPrevName := prevNameToIndex[child.Name]

			if hasPrevName && tree.Children[prevIndex].Merged {
				indexesToOmit[prevIndex] = struct{}{}
			}

			prevNameToIndex[child.Name] = i
		}

		var newChildren []*node.Node

		for i, child := range tree.Children {
			_, shouldOmit := indexesToOmit[i]

			if !shouldOmit {
				newChildren = append(newChildren, child)
			}
		}

		tree.Children = newChildren
	case *node.ListValue:
		for _, listItem := range tree.Children {
			merge(listItem)
		}
	}
}

func MergeMapEntries(tree *node.Node) error {
	merge(tree)

	return nil
}
