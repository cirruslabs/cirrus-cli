package matrix

import (
	"gopkg.in/yaml.v2"
)

// Callback function to be called by traverse().
//
// Traversal continues until this function returns true.
type callback func(item *yaml.MapItem) error

// Implements preorder traversal of the YAML parse tree.
func traverse(item *yaml.MapItem, f callback) error {
	if item.Key != "matrix" {
		// Dig deeper into the tree
		if err := process(item.Value, f); err != nil {
			return err
		}
	}

	// Call the traversal function
	err := f(item)
	if err != nil {
		return err
	}

	return nil
}

// Recursive descent helper for traverse().
func process(something interface{}, f callback) (err error) {
	switch obj := something.(type) {
	case yaml.MapSlice:
		// YAML mapping node
		for i := range obj {
			if err := traverse(&obj[i], f); err != nil {
				return err
			}
		}
	case []interface{}:
		// YAML sequence node
		for _, obj := range obj {
			if err := process(obj, f); err != nil {
				return err
			}
		}
	}

	return
}

// Expands one MapItem if it holds a "matrix" into multiple MapItem's
// Note: this function has side-effects and root will be dirty after the invocation!
func expandIfMatrix(root *yaml.MapItem, item *yaml.MapItem) (result []yaml.MapItem, err error) {
	// Potential "matrix" modifier can only be found in a map
	obj, ok := item.Value.(yaml.MapSlice)
	if !ok {
		return result, nil
	}

	// Split the map into two slices:
	//
	// * beforeSlice contains items that come before the first matrix key
	// * afterSlice contains items that come after the first matrix key
	//
	// Note: we'll only expand the first matrix modifier
	// as a part of this function; the rest will be kept intact
	// and processed by the forthcoming invocations of singlePass().
	var beforeSlice, afterSlice []yaml.MapItem
	var matrix yaml.MapItem

	jumpedOverMatrix := false

	for _, sliceItem := range obj {
		if !jumpedOverMatrix {
			if sliceItem.Key == "matrix" {
				matrix = sliceItem
				jumpedOverMatrix = true
			} else {
				beforeSlice = append(beforeSlice, sliceItem)
			}
		} else {
			afterSlice = append(afterSlice, sliceItem)
		}
	}

	// Keep going deeper if no "matrix" modifiers were found at this level
	if !jumpedOverMatrix {
		return result, nil
	}

	// Extract parametrizations from "matrix" modifier we've selected
	var parametrizations []yaml.MapSlice

	switch obj := matrix.Value.(type) {
	case yaml.MapSlice:
		for _, sliceItem := range obj {
			// Inherit matrix siblings that come before it
			var tmp yaml.MapSlice
			tmp = append(tmp, beforeSlice...)

			// Inject parametrization-specific item from the matrix
			tmp = append(tmp, sliceItem)

			// Inherit matrix siblings that come after it
			tmp = append(tmp, afterSlice...)

			// Generate a single parametrization
			parametrizations = append(parametrizations, tmp)
		}
	case []interface{}:
		for _, listItem := range obj {
			// Inherit matrix siblings that come before it
			var tmp yaml.MapSlice
			tmp = append(tmp, beforeSlice...)

			// Ensure that matrix with a list contains only maps as it's items
			//
			// This restriction was made purely for simplicity's sake and can be lifted in the future.
			innerSlice, ok := listItem.(yaml.MapSlice)
			if !ok {
				return result, ErrMatrixNeedsListOfMaps
			}

			// Inject parametrization-specific items from the matrix
			tmp = append(tmp, innerSlice...)

			// Inherit matrix siblings that come after it
			tmp = append(tmp, afterSlice...)

			// Generate a single parametrization
			parametrizations = append(parametrizations, tmp)
		}
	default:
		// Semantics is undefined for "matrix" modifiers without a collection inside
		return result, ErrMatrixNeedsCollection
	}

	// The Tricky Part™
	//
	// Produces a new diverged root for each parametrization,
	// with a side-effect that the sub-tree of the original root
	// will be overwritten by our parametrization and thus made dirty.
	//
	// However this is fine, because we never re-use the old root anyways
	// and stop processing straight after the parametrization is complete.
	for _, parametrization := range parametrizations {
		item.Value = parametrization

		var divergedRoot yaml.MapItem
		if err := deepcopy(&divergedRoot, *root); err != nil {
			return result, err
		}

		result = append(result, divergedRoot)
	}

	return result, nil
}
