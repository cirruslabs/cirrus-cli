package test_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/commands"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLoadConfiguration(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/configuration")

	command := commands.NewRootCmd()
	command.SetArgs([]string{"internal", "test", "-o simple"})
	err := command.Execute()

	require.Nil(t, err)
}
