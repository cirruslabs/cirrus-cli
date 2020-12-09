package commands_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/commands"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

var validConfigWindows = []byte("windows_container:\n  image: mcr.microsoft.com/windows/servercore:ltsc2019\n\ntask:\n  script: \"(exit 0)\"\n")

// TestRun ensures that the run command can handle the simplest possible configuration without problems.
func TestRun(t *testing.T) {
	testutil.TempChdir(t)

	if err := ioutil.WriteFile(".cirrus.yml", validConfigWindows, 0600); err != nil {
		t.Fatal(err)
	}

	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "-v", "-o simple"})
	err := command.Execute()

	assert.Nil(t, err)
}
