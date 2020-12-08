package executor_test

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
	"github.com/golang/protobuf/ptypes"
	"os"
	"path/filepath"
	"testing"
)

// TestExecutorClone ensures that Executor handles clone instruction properly.
func TestExecutorClone(t *testing.T) {
	dir := testutil.TempDir(t)

	// Create a canary file
	const canaryFile = "canary.file"
	file, err := os.Create(filepath.Join(dir, canaryFile))
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	instance, err := ptypes.MarshalAny(&api.ContainerInstance{
		Image:     "mcr.microsoft.com/windows/servercore:ltsc2019",
		Platform:  api.Platform_WINDOWS,
		OsVersion: "2019",
	})
	if err != nil {
		t.Fatal(err)
	}

	renderer := renderers.NewSimpleRenderer(os.Stdout, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	e, err := executor.New(dir, []*api.Task{
		{
			LocalGroupId: 0,
			Name:         "main",
			Commands: []*api.Command{
				{
					Name: "clone",
					Instruction: &api.Command_CloneInstruction{
						CloneInstruction: &api.CloneInstruction{},
					},
				},
				{
					Name: "check",
					Instruction: &api.Command_ScriptInstruction{
						ScriptInstruction: &api.ScriptInstruction{
							Scripts: []string{fmt.Sprintf("type %s", canaryFile)},
						},
					},
				},
			},
			Instance: instance,
		},
	}, executor.WithContainerBackend(testutil.ContainerBackendFromEnv(t)), executor.WithLogger(logger))
	if err != nil {
		t.Fatal(err)
	}

	if err := e.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}
