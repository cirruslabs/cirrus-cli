package commands_test

import (
	"bytes"
	"github.com/cirruslabs/cirrus-cli/internal/commands"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

// A simplest possible, but valid configuration.
var validConfig = []byte("container:\n  image: debian:latest\ntask:\n  script: true\n")
var validStarlark = []byte(`
def main():
	return {
		'container': {'image': 'debian:latest'},
		'task': {'script': True}
	}
`)

func TestValidateNoArgsNoFile(t *testing.T) {
	testutil.TempChdir(t)

	command := commands.NewRootCmd()
	command.SetArgs([]string{"validate", ""})
	err := command.Execute()

	assert.NotNil(t, err)
}

func TestValidateNoArgsHasFile(t *testing.T) {
	testutil.TempChdir(t)

	if err := ioutil.WriteFile(".cirrus.yml", validConfig, 0600); err != nil {
		t.Fatal(err)
	}

	command := commands.NewRootCmd()
	command.SetArgs([]string{"validate", ""})
	err := command.Execute()

	assert.Nil(t, err)
}

func TestValidateFileArgHasFile(t *testing.T) {
	testutil.TempChdir(t)

	// Craft a simplest possible (but valid) file
	if err := ioutil.WriteFile("custom.yml", validConfig, 0600); err != nil {
		t.Fatal(err)
	}

	command := commands.NewRootCmd()
	command.SetArgs([]string{"validate", "-f", "custom.yml"})
	err := command.Execute()

	assert.Nil(t, err)
}

func TestValidateNoArgsHasFileWithNonStandardExtension(t *testing.T) {
	testutil.TempChdir(t)

	if err := ioutil.WriteFile(".cirrus.yaml", validConfig, 0600); err != nil {
		t.Fatal(err)
	}

	command := commands.NewRootCmd()
	command.SetArgs([]string{"validate", ""})
	err := command.Execute()

	assert.Nil(t, err)
}

func TestValidatePrintFlag(t *testing.T) {
	testutil.TempChdir(t)

	if err := ioutil.WriteFile(".cirrus.star", validStarlark, 0600); err != nil {
		t.Fatal(err)
	}

	buf := bytes.NewBufferString("")

	command := commands.NewRootCmd()
	command.SetArgs([]string{"validate", "--print"})
	command.SetOut(buf)
	err := command.Execute()

	assert.Nil(t, err)
	assert.Contains(t, buf.String(), string(validConfig))
}
