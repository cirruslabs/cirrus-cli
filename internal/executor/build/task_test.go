package build_test

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/build"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestCloneInterception ensures that the clone instruction is replaced with
// a CLI adaptation in the form of a script instruction.
func TestCloneInterception(t *testing.T) {
	task, err := build.NewFromProto(&api.Task{
		Commands: []*api.Command{
			{
				Name: "whatever",
				Instruction: &api.Command_CloneInstruction{
					CloneInstruction: &api.CloneInstruction{},
				},
			},
		},
		Instance: testutil.GetBasicContainerInstance(t, "debian:latest"),
	})
	if err != nil {
		t.Fatal(err)
	}

	require.NotEmpty(t, task.ProtoTask.Commands)
	cloneAdaptation, isScript := task.ProtoTask.Commands[0].Instruction.(*api.Command_ScriptInstruction)
	assert.True(t, isScript)
	assert.NotEmpty(t, cloneAdaptation.ScriptInstruction.Scripts)
}
