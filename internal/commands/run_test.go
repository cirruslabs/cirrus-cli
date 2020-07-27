package commands_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/commands"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
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
