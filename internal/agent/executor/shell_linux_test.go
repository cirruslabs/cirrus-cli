//go:build !windows || !freebsd

package executor

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/agent/environment"
	"testing"
)

func Test_DirectShell_Unix(t *testing.T) {
	testEnv := environment.New(map[string]string{
		"CIRRUS_SHELL": "direct",
	})
	_, output := ShellCommandsAndGetOutput(context.Background(), []string{
		"bash -c 'echo $CIRRUS_SHELL'",
	}, testEnv)

	if output == "direct\n" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", output)
	}
}
