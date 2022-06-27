package node

func (node *Node) ToInterface() (interface{}, error) {
	switch obj := node.Value.(type) {
	case *ScalarValue:
		return obj.Value, nil
	case *ListValue:
		result := []interface{}{}

		for _, child := range node.Children {
			asInterface, err := child.ToInterface()
			if err != nil {
				return nil, err
			}

			result = append(result, asInterface)
		}

		return result, nil
	case *MapValue:
		result := map[string]interface{}{}

		for _, child := range node.Children {
			asInterface, err := child.ToInterface()
			if err != nil {
				return nil, err
			}

			result[child.Name] = asInterface
		}

		return result, nil
	default:
		return nil, node.ParserError("unconvertible node of type %T", node.Value)
	}
}
