package loader

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"go.starlark.net/starlark"
)

var (
	ErrChangesInclude     = errors.New("changes_include() failed")
	ErrChangesIncludeOnly = errors.New("changes_include_only() failed")
)

func generateChangesIncludeBuiltin(affectedFiles []string) *starlark.Builtin {
	result := func(
		thread *starlark.Thread,
		fn *starlark.Builtin,
		args starlark.Tuple,
		kwargs []starlark.Tuple,
	) (starlark.Value, error) {
		rawPatterns, err := starlarkArgsToStrings(args, kwargs)
		if err != nil {
			return nil, err
		}

		count, err := parser.CountMatchingAffectedFiles(affectedFiles, rawPatterns)
		if err != nil {
			return nil, err
		}

		if count > 0 {
			return starlark.True, nil
		}

		return starlark.False, nil
	}

	return starlark.NewBuiltin("changes_include", result)
}

func generateChangesIncludeOnlyBuiltin(affectedFiles []string) *starlark.Builtin {
	result := func(
		thread *starlark.Thread,
		fn *starlark.Builtin,
		args starlark.Tuple,
		kwargs []starlark.Tuple,
	) (starlark.Value, error) {
		rawPatterns, err := starlarkArgsToStrings(args, kwargs)
		if err != nil {
			return nil, err
		}

		count, err := parser.CountMatchingAffectedFiles(affectedFiles, rawPatterns)
		if err != nil {
			return nil, err
		}

		if count > 0 && count == len(affectedFiles) {
			return starlark.True, nil
		}

		return starlark.False, nil
	}

	return starlark.NewBuiltin("changes_include_only", result)
}

func starlarkArgsToStrings(args starlark.Tuple, kwargs []starlark.Tuple) ([]string, error) {
	var result []string

	if len(kwargs) != 0 {
		return nil, fmt.Errorf("%w: found %d keyword arguments, expected 0", ErrChangesInclude, len(kwargs))
	}

	if len(args) == 0 {
		return nil, fmt.Errorf("%w: expected at least 1 positional argument, found 0", ErrChangesInclude)
	}

	for i, arg := range args {
		stringArgument, ok := arg.(starlark.String)
		if !ok {
			return nil, fmt.Errorf("%w: expected %d'th argument to be string, got %s", ErrChangesInclude, i+1,
				arg.Type())
		}

		result = append(result, stringArgument.GoString())
	}

	return result, nil
}
