package executor

import (
	"fmt"
)

func EnvMapAsSlice(env map[string]string) []string {
	var result []string

	for key, value := range env {
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}

	return result
}
