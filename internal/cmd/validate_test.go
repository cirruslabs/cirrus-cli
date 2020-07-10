package cmd_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/cmd"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

// A simplest possible, but valid configuration.
var validConfig = []byte("task:\n  script: true")

// tempDir supplements an alternative to TB.TempDir()[1], which is only available in 1.15.
// [1]: https://github.com/golang/go/issues/35998
func tempDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	})

	return dir
}

// tempChdir switches to a temporary per-test directory.
func tempChdir(t *testing.T) {
	if err := os.Chdir(tempDir(t)); err != nil {
		t.Fatal(err)
	}
}

func TestValidateNoArgsNoFile(t *testing.T) {
	tempChdir(t)

	command := cmd.NewRootCmd()
	command.SetArgs([]string{"validate", ""})
	err := command.Execute()

	assert.NotNil(t, err)
}

func TestValidateNoArgsHasFile(t *testing.T) {
	tempChdir(t)

	if err := ioutil.WriteFile(".cirrus.yml", validConfig, 0600); err != nil {
		t.Fatal(err)
	}

	command := cmd.NewRootCmd()
	command.SetArgs([]string{"validate", ""})
	err := command.Execute()

	assert.Nil(t, err)
}

func TestValidateFileArgHasFile(t *testing.T) {
	tempChdir(t)

	// Craft a simplest possible (but valid) file
	if err := ioutil.WriteFile("custom.yml", validConfig, 0600); err != nil {
		t.Fatal(err)
	}

	command := cmd.NewRootCmd()
	command.SetArgs([]string{"validate", "-f", "custom.yml"})
	err := command.Execute()

	assert.Nil(t, err)
}
