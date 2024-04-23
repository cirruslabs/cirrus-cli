package executor

import (
	"bytes"
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/agent/environment"
)

func ShellCommandsAndGetOutput(
	ctx context.Context,
	scripts []string,
	custom_env *environment.Environment,
) (bool, string) {
	var buffer bytes.Buffer
	cmd, err := ShellCommandsAndWait(ctx, scripts, custom_env, func(bytes []byte) (int, error) {
		return buffer.Write(bytes)
	}, false)
	return err == nil && cmd.ProcessState.Success(), buffer.String()
}
