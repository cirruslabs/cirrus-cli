package matrix

import (
	"errors"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"strings"
)

// errNoExpansionDone is returned when a single pass yields no matrix expansions.
var errNoExpansionDone = errors.New("no matrix expansion was done")

func singlePass(inputTree *node.Node) error {
	// Find a matrix node to expand
	matrixNode := inputTree.DeepFindChild("matrix")
	if matrixNode == nil {
		return errNoExpansionDone
	}

	// Ensure this matrix node is attached to either a task or a docker_builder
	taskNode := matrixNode.FindParent(func(nodeName string) bool {
		return strings.HasSuffix(nodeName, "task") || strings.HasSuffix(nodeName, "docker_builder")
	})
	if taskNode == nil {
		return matrixNode.ParserError("matrix can be defined only under a task or docker_builder")
	}

	var newTasks []*node.Node

	switch matrixNode.Value.(type) {
	case *node.MapValue:
		for _, child := range matrixNode.Children {
			newTasks = append(newTasks, taskNode.DeepCopyWithReplacements(matrixNode, []*node.Node{child}))
		}
	case *node.ListValue:
		for _, child := range matrixNode.Children {
			if _, ok := child.Value.(*node.MapValue); !ok {
				return child.ParserError("matrix with a list can only contain maps as it's items")
			}

			newTasks = append(newTasks, taskNode.DeepCopyWithReplacements(matrixNode, child.Children))
		}
	default:
		return matrixNode.ParserError("matrix should contain a collection")
	}

	taskNode.ReplaceWith(newTasks)

	return nil
}

func ExpandMatrices(tree *node.Node) error {
	for {
		if err := singlePass(tree); err != nil {
			if errors.Is(err, errNoExpansionDone) {
				return nil
			}

			return err
		}
	}
}
