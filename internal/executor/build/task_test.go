package build_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/build"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestCloneInterception ensures that the first command named "clone" is removed.
func TestCloneInterception(t *testing.T) {
	examples := map[string][]*api.Command{
		"clone command with clone instruction": {
			{
				Name: "clone",
				Instruction: &api.Command_CloneInstruction{
					CloneInstruction: &api.CloneInstruction{},
				},
			},
		},
		"clone command with script instruction": {
			{
				Name: "clone",
				Instruction: &api.Command_ScriptInstruction{
					ScriptInstruction: &api.ScriptInstruction{
						Scripts: []string{"git clone --help"},
					},
				},
			},
		},
		"clone command with no instruction": {
			{
				Name: "clone",
			},
		},
	}

	for exampleName, commands := range examples {
		commands := commands
		t.Run(exampleName, func(t *testing.T) {
			task, err := build.NewFromProto(&api.Task{
				Commands: commands,
				Instance: testutil.GetBasicContainerInstance(t, "debian:latest"),
			}, nil)
			if err != nil {
				t.Fatal(err)
			}

			assert.Empty(t, task.Commands)
		})
	}
}
