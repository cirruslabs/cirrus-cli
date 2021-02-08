package parallels

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var (
	ErrParallelsCommandNotFound = errors.New("Parallels command not found")
	ErrParallelsCommandNonZero  = errors.New("Parallels command returned non-zero exit code")
)

func runParallelsCommand(ctx context.Context, commandName string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, commandName, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", "", fmt.Errorf("%w: %q not found in PATH, make sure Parallels is installed",
				ErrParallelsCommandNotFound, commandName)
		}

		if _, ok := err.(*exec.ExitError); ok {
			// Parallels command failed, redefine the error
			// to be the Parallels-specific output
			err = fmt.Errorf("%w: %q", ErrParallelsCommandNonZero, firstNonEmptyLine(stderr.String(), stdout.String()))
		}
	}

	return stdout.String(), stderr.String(), err
}

func Prlctl(ctx context.Context, args ...string) (string, string, error) {
	return runParallelsCommand(ctx, "prlctl", args...)
}

func Prlsrvctl(ctx context.Context, args ...string) (string, string, error) {
	return runParallelsCommand(ctx, "prlsrvctl", args...)
}

func firstNonEmptyLine(outputs ...string) string {
	for _, output := range outputs {
		for _, line := range strings.Split(output, "\n") {
			if line != "" {
				return line
			}
		}
	}

	return ""
}
