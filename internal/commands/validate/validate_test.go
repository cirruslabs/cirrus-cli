package validate_test

import (
	"bytes"
	"github.com/cirruslabs/cirrus-cli/internal/commands"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

func TestValidateCloudInstance(t *testing.T) {
	testutil.TempChdirPopulatedWith(t, "testdata/cloud-instance")

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	command := commands.NewRootCmd()
	command.SetArgs([]string{"validate"})
	command.SetOut(writer)
	command.SetErr(writer)
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}

	assert.NotContains(t, buf.String(), "failed",
		"additional instance should be fetched and transformed successfully")
}
