package tart

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/cirruslabs/cirrus-cli/pkg/privdrop"
	"github.com/cirruslabs/echelon"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
	"go.uber.org/zap/zapio"
)

const tartCommandName = "tart"

var (
	ErrTartNotFound = errors.New("tart command not found")
	ErrTartFailed   = errors.New("tart command returned non-zero exit code")
)

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
	_, err := exec.LookPath(tartCommandName)

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
	ctx, span := tracer.Start(ctx, fmt.Sprintf("exec-command/%s-%s", tartCommandName, name))
	defer span.End()

	args = append([]string{name}, args...)

	cmd := exec.CommandContext(ctx, tartCommandName, args...)

	// Drop privileges for the spawned process, if requested
	if sysProcAttr := privdrop.SysProcAttr; sysProcAttr != nil {
		cmd.SysProcAttr = sysProcAttr
	}

	// Work around https://github.com/golang/go/issues/23019,
	// most likely happens when running with --net-softnet
	cmd.WaitDelay = time.Second

	// Default environment
	cmd.Env = cmd.Environ()

	// Additional environment
	for key, value := range additionalEnvironment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Propagate current W3C Trace Context to a Tart binary using the environment variables[1]
	//
	// [1]: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/context/env-carriers.md
	mapCarrier := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(ctx, mapCarrier)
	for key, value := range mapCarrier {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", strings.ToUpper(key), value))
	}

	var stdout, stderr bytes.Buffer

	zapFields := []zap.Field{
		// "Field value of type context.Context is used as context when emitting log records."[1]
		//
		// [1]: https://pkg.go.dev/go.opentelemetry.io/contrib/bridges/otelzap
		zap.Any("context", ctx),
		zap.String("command", strings.Join(append([]string{cmd.Path}, cmd.Args...), " ")),
	}

	zapInfoWriter := &zapio.Writer{
		Log:   zap.L().With(zapFields...),
		Level: zap.InfoLevel,
	}
	defer zapInfoWriter.Close()

	zapErrorWriter := &zapio.Writer{
		Log:   zap.L().With(zapFields...),
		Level: zap.WarnLevel,
	}
	defer zapErrorWriter.Close()

	cmd.Stdout = io.MultiWriter(&stdout, &loggerAsWriter{
		level:    echelon.InfoLevel,
		delegate: logger,
	}, zapInfoWriter)
	cmd.Stderr = io.MultiWriter(&stderr, &loggerAsWriter{
		level:    echelon.WarnLevel,
		delegate: logger,
	}, zapErrorWriter)

	err := cmd.Run()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", "", fmt.Errorf("%w: %s command not found in PATH, make sure Tart is installed",
				ErrTartNotFound, tartCommandName)
		}

		if _, ok := err.(*exec.ExitError); ok {
			// Tart command failed, redefine the error
			// to be the Tart-specific output
			err = fmt.Errorf("%w: %q", ErrTartFailed, firstNonEmptyLine(stderr.String(), stdout.String()))
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
