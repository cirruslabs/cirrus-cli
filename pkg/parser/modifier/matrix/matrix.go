package matrix

import (
	"errors"
	"gopkg.in/yaml.v2"
	"strings"
)

// ErrMatrixNeedsCollection is returned when the matrix modifier
// does not contain a collection (either map or slice) inside.
var ErrMatrixNeedsCollection = errors.New("matrix should contain a collection")

// ErrMatrixNeedsListOfMaps is returned when the matrix modifier contains
// something other than maps (e.g. lists or scalars) as it's items.
var ErrMatrixNeedsListOfMaps = errors.New("matrix with a list can only contain maps as it's items")

// Recursively processes each "outer" map key of the loaded YAML document
// in an attempt to produce multiple keys as a result of matrix expansion.
func singlePass(inputTree yaml.MapSlice) (yaml.MapSlice, error) {
	var outputTree yaml.MapSlice

	if len(inputTree) == 0 {
		return nil, nil
	}

	for i := range inputTree {
		var treeToExpand yaml.MapItem
		// deepcopy since expandIfMatrix has side effects
		if err := deepcopy(&treeToExpand, inputTree[i]); err != nil {
			return nil, err
		}

		// Ensure that <>
		if !strings.HasSuffix(inputTree[i].Key.(string), "task") &&
			!strings.HasSuffix(inputTree[i].Key.(string), "docker_builder") {
			outputTree = append(outputTree, treeToExpand)
			continue
		}

		var expandedTrees []yaml.MapItem
		expandedTreesCollector := func(item *yaml.MapItem) (bool, error) {
			newTrees, expandErr := expandIfMatrix(&treeToExpand, item)
			// stop once found any expansion
			if len(newTrees) != 0 {
				expandedTrees = newTrees
				return true, nil
			}
			return false, expandErr
		}

		if err := traverse(&treeToExpand, expandedTreesCollector); err != nil {
			return nil, err
		}

		if len(expandedTrees) == 0 {
			outputTree = append(outputTree, treeToExpand)
		} else {
			outputTree = append(outputTree, expandedTrees...)
		}
	}

	return outputTree, nil
}

func ExpandMatrices(tree yaml.MapSlice) (yaml.MapSlice, error) {
	for {
		expandedTree, err := singlePass(tree)
		if err != nil {
			return nil, err
		}

		if len(expandedTree) != len(tree) {
			// Update tree
			tree = expandedTree
		} else {
			// Consider the preprocessing done once singlePass() stops expanding the document
			// (which means no "matrix" modifier was found)
			break
		}
	}

	return tree, nil
}
