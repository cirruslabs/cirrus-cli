package parallels_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/parallels"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestPrlctl(t *testing.T) {
	ctx := context.Background()

	_, imageOk := os.LookupEnv("CIRRUS_INTERNAL_PARALLELS_VM")
	_, userOk := os.LookupEnv("CIRRUS_INTERNAL_PARALLELS_SSH_USER")
	_, passwordOk := os.LookupEnv("CIRRUS_INTERNAL_PARALLELS_SSH_PASSWORD")
	if !imageOk || !userOk || !passwordOk {
		t.SkipNow()
	}

	// Working example
	_, _, err := parallels.Prlctl(ctx, "list")
	assert.NoError(t, err)

	// Non-working example
	expectedErrorMessage := `Parallels command returned non-zero exit code: "Failed to get VM config: The virtual` +
		` machine could not be found. The virtual machine is not registered in the virtual machine directory on your Mac."`

	_, _, err = parallels.Prlctl(ctx, "list", "non-existent-vm")
	assert.Error(t, err)
	assert.Equal(t, expectedErrorMessage, err.Error())
}
