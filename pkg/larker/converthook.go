package larker

import (
	"errors"
	"fmt"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

var ErrStarlarkConversion = errors.New("failed to convert Starlark data type")

func starlarkValueAsInterface(value starlark.Value) (interface{}, error) {
	switch v := value.(type) {
	case starlark.NoneType:
		return nil, nil
	case starlark.Bool:
		return bool(v), nil
	case starlark.Int:
		res, _ := v.Int64()

		return res, nil
	case starlark.Float:
		return float64(v), nil
	case starlark.String:
		return string(v), nil
	case *starlark.List:
		it := v.Iterate()
		defer it.Done()

		var listItem starlark.Value
		var result []interface{}

		for it.Next(&listItem) {
			listItemInterfaced, err := starlarkValueAsInterface(listItem)
			if err != nil {
				return nil, err
			}

			result = append(result, listItemInterfaced)
		}

		return result, nil
	case *starlark.Dict:
		result := map[string]interface{}{}

		for _, item := range v.Items() {
			key := item[0]
			value := item[1]

			if _, ok := key.(*starlark.String); !ok {
				return nil, fmt.Errorf("%w: all dict keys should be strings", ErrStarlarkConversion)
			}

			dictValueInterfaced, err := starlarkValueAsInterface(value)
			if err != nil {
				return nil, err
			}

			result[key.String()] = dictValueInterfaced
		}

		return result, nil
	default:
		return nil, fmt.Errorf("%w: unsupported type %T", ErrStarlarkConversion, value)
	}
}

func interfaceAsStarlarkValue(value interface{}) (starlark.Value, error) {
	switch v := value.(type) {
	case nil:
		return starlark.None, nil
	case bool:
		return starlark.Bool(v), nil
	case int:
		return starlark.MakeInt(v), nil
	case int64:
		return starlark.MakeInt64(v), nil
	case uint:
		return starlark.MakeUint(v), nil
	case uint64:
		return starlark.MakeUint64(v), nil
	case float32:
		return starlark.Float(v), nil
	case float64:
		return starlark.Float(v), nil
	case string:
		return starlark.String(v), nil
	case []interface{}:
		result := starlark.NewList([]starlark.Value{})

		for _, item := range v {
			listValueStarlarked, err := interfaceAsStarlarkValue(item)
			if err != nil {
				return nil, err
			}

			if err := result.Append(listValueStarlarked); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrStarlarkConversion, err)
			}
		}

		return result, nil
	case map[string]interface{}:
		result := map[string]starlark.Value{}

		for key, value := range v {
			mapValueStarlarked, err := interfaceAsStarlarkValue(value)
			if err != nil {
				return nil, err
			}

			result[key] = mapValueStarlarked
		}

		dict := starlarkstruct.FromStringDict(starlarkstruct.Default, result)

		return dict, nil
	default:
		return nil, fmt.Errorf("%w: unsupported type %T", ErrStarlarkConversion, value)
	}
}
