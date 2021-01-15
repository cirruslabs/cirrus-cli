package expander

import (
	"fmt"
	"sort"
	"strings"
)

const defaultMaxExpansionIterations = 10

type expander struct {
	maxExpansionIterations int
	precise                bool
}

func ExpandEnvironmentVariables(s string, env map[string]string, opts ...Option) string {
	e := &expander{
		maxExpansionIterations: defaultMaxExpansionIterations,
	}

	for _, opt := range opts {
		opt(e)
	}

	// Create a sorted view of the environment map keys to expand the longest keys first
	// and avoid cases where "$CIRRUS_BRANCH" is expanded into "true_BRANCH", assuming
	// env = map[string]string{"CIRRUS": "true", "CIRRUS_BRANCH": "main"})
	var sortedKeys []string
	for key := range env {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return !(sortedKeys[i] < sortedKeys[j])
	})

	for i := 0; i < e.maxExpansionIterations; i++ {
		beforeExpansion := s

		for _, key := range sortedKeys {
			s = strings.ReplaceAll(s, fmt.Sprintf("$%s", key), env[key])
			if e.precise && beforeExpansion != s {
				break
			}

			s = strings.ReplaceAll(s, fmt.Sprintf("${%s}", key), env[key])
			if e.precise && beforeExpansion != s {
				break
			}

			s = strings.ReplaceAll(s, fmt.Sprintf("%%%s%%", key), env[key])
			if e.precise && beforeExpansion != s {
				break
			}
		}

		// Don't wait till the end of the loop if we are not progressing
		if s == beforeExpansion {
			break
		}
	}

	return s
}
