package executor_test

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"testing"
)

// TestExecutorEmpty ensures that Executor works fine with an empty task list.
func TestExecutorEmpty(t *testing.T) {
	dir := testutil.TempDir(t)

	e, err := executor.New(dir, []*api.Task{})
	if err != nil {
		t.Fatal(err)
	}

	if err := e.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}

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
							Scripts: []string{fmt.Sprintf("test -e %s", canaryFile)},
						},
					},
				},
			},
			Instance: testutil.GetBasicContainerInstance(t, "debian:latest"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := e.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// TestExecutorScript ensures that Executor can run a few simple commands.
func TestExecutorScript(t *testing.T) {
	dir := testutil.TempDir(t)

	logger := logrus.New()
	logger.Level = logrus.TraceLevel

	e, err := executor.New(dir, []*api.Task{
		{
			LocalGroupId: 0,
			Name:         "mainTask",
			Commands: []*api.Command{
				{
					Name: "firstCommand",
					Instruction: &api.Command_ScriptInstruction{
						ScriptInstruction: &api.ScriptInstruction{
							Scripts: []string{
								"date",
							},
						},
					},
				},
				{
					Name: "secondCommand",
					Instruction: &api.Command_ScriptInstruction{
						ScriptInstruction: &api.ScriptInstruction{
							Scripts: []string{
								"uname -a",
							},
						},
					},
				},
			},
			Instance: testutil.GetBasicContainerInstance(t, "debian:latest"),
		},
	}, executor.WithLogger(logger))
	if err != nil {
		t.Fatal(err)
	}

	if err := e.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}
