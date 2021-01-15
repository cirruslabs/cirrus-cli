package expander_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/expander"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExpandEnvironmentVariablesIsDeterministic(t *testing.T) {
	env := map[string]string{
		"C":             "not deterministic",
		"CI":            "not deterministic",
		"CIR":           "not deterministic",
		"CIRR":          "not deterministic",
		"CIRRU":         "not deterministic",
		"CIRRUS":        "true",
		"CIRRUS_BRANCH": "main",
		"CIRRUS_":       "not deterministic",
		"CIRRUS_B":      "not deterministic",
		"CIRRUS_BR":     "not deterministic",
		"CIRRUS_BRA":    "not deterministic",
		"CIRRUS_BRAN":   "not deterministic",
		"CIRRUS_BRANC":  "not deterministic",
	}

	assert.Equal(t, "main", expander.ExpandEnvironmentVariables("$CIRRUS_BRANCH", env))

	assert.Equal(t, "true main", expander.ExpandEnvironmentVariables("$CIRRUS $CIRRUS_BRANCH", env))
	assert.Equal(t, "main true", expander.ExpandEnvironmentVariables("$CIRRUS_BRANCH $CIRRUS", env))
}
