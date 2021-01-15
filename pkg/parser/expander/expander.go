package expander

import (
	"fmt"
	"sort"
	"strings"
)

func ExpandEnvironmentVariables(s string, env map[string]string) string {
	const maxExpansionIterations = 10

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

	for i := 0; i < maxExpansionIterations; i++ {
		beforeExpansion := s

		for _, key := range sortedKeys {
			s = strings.ReplaceAll(s, fmt.Sprintf("$%s", key), env[key])
			s = strings.ReplaceAll(s, fmt.Sprintf("${%s}", key), env[key])
			s = strings.ReplaceAll(s, fmt.Sprintf("%%%s%%", key), env[key])
		}

		// Don't wait till the end of the loop if we are not progressing
		if s == beforeExpansion {
			break
		}
	}

	return s
}
