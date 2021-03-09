package loader

import (
	"errors"
	"fmt"
	glob "github.com/cirruslabs/go-java-glob"
	"go.starlark.net/starlark"
)

var ErrChangesInclude = errors.New("changes_include() failed")

func generateChangesIncludeBuiltin(affectedFiles []string) *starlark.Builtin {
	changesInclude := func(
		thread *starlark.Thread,
		fn *starlark.Builtin,
		args starlark.Tuple,
		kwargs []starlark.Tuple,
	) (starlark.Value, error) {
		var rawPatterns []string

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

			rawPatterns = append(rawPatterns, stringArgument.GoString())
		}

		for _, rawPattern := range rawPatterns {
			pattern, err := glob.ToRegexPattern(rawPattern, false)
			if err != nil {
				return nil, err
			}

			for _, affectedFile := range affectedFiles {
				if pattern.MatchString(affectedFile) {
					return starlark.True, nil
				}
			}
		}

		return starlark.False, nil
	}

	return starlark.NewBuiltin("changes_include", changesInclude)
}
