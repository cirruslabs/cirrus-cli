package matrix

import (
	"errors"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"strings"
)

// ErrMatrixNeedsCollection is returned when the matrix modifier
// does not contain a collection (either map or slice) inside.
var ErrMatrixNeedsCollection = errors.New("matrix should contain a collection")

// ErrMatrixNeedsListOfMaps is returned when the matrix modifier contains
// something other than maps (e.g. lists or scalars) as it's items.
var ErrMatrixNeedsListOfMaps = errors.New("matrix with a list can only contain maps as it's items")

// ErrMatrixIsMisplaced is returned when the matrix modifier is attached
// to a task type other than task or docker_builder.
var ErrMatrixIsMisplaced = errors.New("matrix can be defined only under a task or docker_builder")

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
		return ErrMatrixIsMisplaced
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
				return ErrMatrixNeedsListOfMaps
			}

			newTasks = append(newTasks, taskNode.DeepCopyWithReplacements(matrixNode, child.Children))
		}
	default:
		return ErrMatrixNeedsCollection
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
