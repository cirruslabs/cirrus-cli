package commands_test

import (
	"bytes"
	"github.com/cirruslabs/cirrus-cli/internal/commands"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestRun(t *testing.T) {
	testutil.TempChdir(t)

	if err := ioutil.WriteFile(".cirrus.yml", validConfig, 0600); err != nil {
		t.Fatal(err)
	}

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", ""})
	err := command.Execute()

	assert.Nil(t, err)
}

// TestRunTaskFiltering ensures that the task filtering mechanism only runs the task specified by the user.
func TestRunTaskFiltering(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/run-task-filtering")

	var examples = map[string]struct {
		Pattern         string
		ExpectedStrings []string
	}{
		"first single task": {"first_working", []string{
			"task first_working (0) succeeded",
		}},
		"second single task": {"Second Working", []string{
			"task Second Working (2) succeeded",
		}},
		"first task case insensitivity": {"FiRsT_WoRkInG", []string{
			"task first_working (0) succeeded",
		}},
		"second task case insensitivity": {"SECOND WORKING", []string{
			"task Second Working (2) succeeded",
		}},
	}

	for exampleName, example := range examples {
		example := example
		t.Run(exampleName, func(t *testing.T) {
			// Create os.Stderr writer that duplicates it's output to buf
			buf := bytes.NewBufferString("")
			writer := io.MultiWriter(os.Stderr, buf)

			command := commands.NewRootCmd()
			command.SetArgs([]string{"run", example.Pattern})
			command.SetOut(writer)
			command.SetErr(writer)
			err := command.Execute()

			require.Nil(t, err)
			for _, expectedString := range example.ExpectedStrings {
				require.Contains(t, buf.String(), expectedString)
			}
		})
	}
}

// TestRunTaskDependencyRemoval ensures that dependencies for the task
// selected by the task filtering mechanism are removed properly.
func TestRunTaskDependencyRemoval(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/run-task-dependency-removal")

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "bar"})
	err := command.Execute()

	require.Nil(t, err)
}
