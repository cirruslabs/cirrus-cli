package vetu

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/cirruslabs/echelon"
	"go.uber.org/zap"
	"go.uber.org/zap/zapio"
)

const vetuCommandName = "vetu"

var (
	ErrVetuNotFound = errors.New("vetu command not found")
	ErrVetuFailed   = errors.New("vetu command returned non-zero exit code")
)

type CmdOpts struct {
	AdditionalEnvironment map[string]string
	Logger                *echelon.Logger
	StandardOutputToLogs  bool
}

type loggerAsWriter struct {
	level    echelon.LogLevel
	delegate *echelon.Logger
}

func (l loggerAsWriter) Write(p []byte) (n int, err error) {
	if l.delegate != nil {
		l.delegate.Logf(l.level, "%s", strings.TrimSpace(string(p)))
	}
	return len(p), nil
}

func Installed() bool {
	_, err := exec.LookPath(vetuCommandName)

	return err == nil
}

func Cmd(
	ctx context.Context,
	additionalEnvironment map[string]string,
	name string,
	args ...string,
) (string, string, error) {
	return CmdWithLogger(ctx, additionalEnvironment, nil, name, args...)
}

func CmdWithLogger(
	ctx context.Context,
	additionalEnvironment map[string]string,
	logger *echelon.Logger,
	name string,
	args ...string,
) (string, string, error) {
	return CmdWithOpts(ctx, CmdOpts{
		AdditionalEnvironment: additionalEnvironment,
		Logger:                logger,
	}, name, args...)
}

func CmdWithOpts(
	ctx context.Context,
	opts CmdOpts,
	name string,
	args ...string,
) (string, string, error) {
	ctx, span := tracer.Start(ctx, fmt.Sprintf("exec-command/%s-%s", vetuCommandName, name))
	defer span.End()

	args = append([]string{name}, args...)

	cmd := exec.CommandContext(ctx, vetuCommandName, args...)

	// Default environment
	cmd.Env = cmd.Environ()

	// Additional environment
	for key, value := range opts.AdditionalEnvironment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	var stdout, stderr bytes.Buffer

	zapFields := []zap.Field{
		// "Field value of type context.Context is used as context when emitting log records."[1]
		//
		// [1]: https://pkg.go.dev/go.opentelemetry.io/contrib/bridges/otelzap
		zap.Any("context", ctx),
		zap.String("command", strings.Join(append([]string{cmd.Path}, cmd.Args...), " ")),
	}

	stdoutWriters := []io.Writer{
		&stdout,
		&loggerAsWriter{
			level:    echelon.InfoLevel,
			delegate: opts.Logger,
		},
	}

	if opts.StandardOutputToLogs {
		zapInfoWriter := &zapio.Writer{
			Log:   zap.L().With(zapFields...),
			Level: zap.InfoLevel,
		}
		defer zapInfoWriter.Close()

		stdoutWriters = append(stdoutWriters, zapInfoWriter)
	}

	zapErrorWriter := &zapio.Writer{
		Log:   zap.L().With(zapFields...),
		Level: zap.WarnLevel,
	}
	defer zapErrorWriter.Close()

	cmd.Stdout = io.MultiWriter(stdoutWriters...)
	cmd.Stderr = io.MultiWriter(&stderr, &loggerAsWriter{
		level:    echelon.WarnLevel,
		delegate: opts.Logger,
	}, zapErrorWriter)

	err := cmd.Run()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", "", fmt.Errorf("%w: %s command not found in PATH, make sure Vetu is installed",
				ErrVetuNotFound, vetuCommandName)
		}

		if _, ok := err.(*exec.ExitError); ok {
			// Vetu command failed, redefine the error
			// to be the Vetu-specific output
			err = fmt.Errorf("%w: %q", ErrVetuFailed, firstNonEmptyLine(stderr.String(), stdout.String()))
		}
	}

	return stdout.String(), stderr.String(), err
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
