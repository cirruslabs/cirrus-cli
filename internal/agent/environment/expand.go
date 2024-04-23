package environment

import (
	"maps"
	"os"
	"regexp"
	"strings"
)

func ExpandEnvironmentRecursively(environment map[string]string) map[string]string {
	result := maps.Clone(environment)

	for step := 0; step < 10; step++ {
		var changed = false
		for key, value := range result {
			originalValue := result[key]
			expandedValue := expandTextOSFirst(value, result)

			selfRecursion := strings.Contains(expandedValue, "$"+key) ||
				strings.Contains(expandedValue, "${"+key) ||
				strings.Contains(expandedValue, "%"+key)
			if selfRecursion {
				// detected self-recursion
				continue
			}

			result[key] = expandedValue

			if originalValue != expandedValue {
				changed = true
			}
		}

		if !changed {
			break
		}
	}
	return result
}

func (env *Environment) ExpandText(text string) string {
	return expandTextExtended(text, func(name string) (string, bool) {
		if userValue, ok := env.Lookup(name); ok {
			return userValue, true
		}

		return os.LookupEnv(name)
	})
}

func expandTextOSFirst(text string, customEnv map[string]string) string {
	return expandTextExtended(text, func(name string) (string, bool) {
		if osValue, ok := os.LookupEnv(name); ok {
			return osValue, true
		}
		userValue, ok := customEnv[name]
		return userValue, ok
	})
}

func expandTextExtended(text string, lookup func(string) (string, bool)) string {
	var re = regexp.MustCompile(`%(\w+)%`)
	return os.Expand(re.ReplaceAllString(text, `${$1}`), func(text string) string {
		parts := strings.SplitN(text, ":", 2)

		name := parts[0]
		defaultValue := ""
		if len(parts) > 1 {
			defaultValue = parts[1]
		}

		if value, ok := lookup(name); ok {
			return value
		}

		return defaultValue
	})
}
