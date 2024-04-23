package environment_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/environment"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMergeNonSensitive(t *testing.T) {
	env := environment.New(map[string]string{
		"NON_SENSITIVE_VALUE": "shouldn't be masked",
	})

	env.Merge(map[string]string{"SOME_VALUE": "shouldn't be masked"}, false)

	assert.Equal(t, []string{}, env.SensitiveValues())
}

func TestMergeSensitive(t *testing.T) {
	env := environment.New(map[string]string{
		"NON_SENSITIVE_VALUE": "shouldn't be masked",
	})

	env.Merge(map[string]string{"SOME_VALUE": "SHOULD be masked"}, true)

	assert.Equal(t, []string{"SHOULD be masked"}, env.SensitiveValues())
}

func TestWellKnownSensitiveVariables(t *testing.T) {
	env := environment.New(map[string]string{
		"NON_SENSITIVE":  "shouldn't be masked",
		"IS_TOKEN":       "SHOULD be masked",
		"IS_EMPTY_TOKEN": "",
	})

	assert.Equal(t, []string{"SHOULD be masked"}, env.SensitiveValues())
}
