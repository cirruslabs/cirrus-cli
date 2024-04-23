package executor_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestDeduplicatePaths(t *testing.T) {
	testCases := []struct {
		Name           string
		Input          []string
		ExpectedOutput []string
	}{
		{
			Name: "simple",
			Input: []string{
				"/tmp/node_modules/module/node_modules",
				"/tmp/node_modules",
			},
			ExpectedOutput: []string{
				"/tmp/node_modules",
			},
		},
		{
			Name: "path-aware comparison",
			Input: []string{
				"/tmp/node_modules/module/node_modules",
				"/tmp/node_mod",
			},
			ExpectedOutput: []string{
				"/tmp/node_mod",
				"/tmp/node_modules/module/node_modules",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.Name, func(t *testing.T) {
			var osAwareInput []string
			for _, input := range testCase.Input {
				osAwareInput = append(osAwareInput, filepath.FromSlash(input))
			}

			var osAwareExpectedOutput []string
			for _, expectedOutput := range testCase.ExpectedOutput {
				osAwareExpectedOutput = append(osAwareExpectedOutput, filepath.FromSlash(expectedOutput))
			}

			assert.Equal(t, osAwareExpectedOutput, executor.DeduplicatePaths(osAwareInput))
		})
	}
}
