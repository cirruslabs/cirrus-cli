package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/local"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/loader"
	"github.com/spf13/cobra"
	"go.starlark.net/starlark"
)

var ErrEval = errors.New("eval failed")

func newEvalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eval <script.star|->",
		Short: "Evaluate a Starlark script and stream print output",
		Long: `Evaluate a Starlark script in top-level mode without requiring main().

This command is a lightweight, LLM-friendly way to run Python-like scripts.
Instead of starting a full Python interpreter, it executes Starlark directly
and exposes Cirrus built-ins that are useful for automation and data handling.

Load built-ins like this:
  load("cirrus", "http", "fs", "json", "yaml")

Output is produced only by print(...) and println(...).`,
		Example: strings.TrimSpace(`
cirrus eval scripts/task.star
cat script.star | cirrus eval -
cat <<'STAR' | cirrus eval -
load("cirrus", "http")
status = http.get("https://www.githubstatus.com/api/v2/status.json").json()
print(status["status"]["description"])
STAR
`),
		Args: cobra.ExactArgs(1),
		RunE: eval,
	}

	return cmd
}

func eval(cmd *cobra.Command, args []string) error {
	// https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	source, filename, err := readEvalSource(cmd, args[0])
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEval, err)
	}

	if err := runTopLevelStarlark(cmd.Context(), source, filename, cmd.OutOrStdout()); err != nil {
		var evalErr *starlark.EvalError
		if errors.As(err, &evalErr) {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), evalErr.Backtrace())
		}

		return fmt.Errorf("%w: %v", ErrEval, err)
	}

	return nil
}

func readEvalSource(cmd *cobra.Command, sourcePath string) (source string, filename string, err error) {
	if sourcePath == "-" {
		sourceBytes, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", "", err
		}

		return string(sourceBytes), "<stdin>", nil
	}

	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", "", err
	}

	return string(sourceBytes), sourcePath, nil
}

func runTopLevelStarlark(ctx context.Context, source, filename string, output io.Writer) error {
	processEnvironment := processEnvironment()

	lfs := local.New(".")
	moduleLoader := loader.NewLoader(ctx, lfs, processEnvironment, []string{}, false, nil)

	thread := &starlark.Thread{
		Load: moduleLoader.LoadFunc(lfs),
		Print: func(thread *starlark.Thread, msg string) {
			_, _ = fmt.Fprintln(output, msg)
		},
	}

	predeclared := starlark.StringDict{
		"println": starlark.Universe["print"],
	}

	errCh := make(chan error, 1)

	go func() {
		_, err := starlark.ExecFile(thread, filename, source, predeclared)
		errCh <- err
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		thread.Cancel(ctx.Err().Error())
		return ctx.Err()
	}
}

func processEnvironment() map[string]string {
	result := make(map[string]string)

	for _, rawEnvVar := range os.Environ() {
		envVarParts := strings.SplitN(rawEnvVar, "=", 2)
		if len(envVarParts) != 2 {
			continue
		}

		result[envVarParts[0]] = envVarParts[1]
	}

	return result
}
