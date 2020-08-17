package commands

import (
	"errors"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	"github.com/cirruslabs/cirrus-cli/internal/executor/taskfilter"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var ErrRun = errors.New("run failed")

var dirty bool
var environment []string
var runFile string
var verbose bool

// envArgsToMap parses and expands environment arguments like "A=B" (set operation)
// and "A" (pass-through operation) into a map suitable for use across the codebase.
func envArgsToMap(arguments []string) map[string]string {
	result := make(map[string]string)

	const (
		keyPart = iota
		valuePart
		maxParts
	)

	for _, argument := range arguments {
		parts := strings.SplitN(argument, "=", maxParts)

		if len(parts) == maxParts {
			// Set mode: simply assign the provided value to key
			result[parts[keyPart]] = parts[valuePart]
		} else {
			// Pass-through mode: resolve the value for the provided key and assign it (if any)
			resolvedValue, found := os.LookupEnv(parts[keyPart])
			if !found {
				break
			}
			result[parts[keyPart]] = resolvedValue
		}
	}

	return result
}

func run(cmd *cobra.Command, args []string) error {
	// https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	envMap := envArgsToMap(environment)

	// Parse
	p := parser.Parser{Environment: envMap}
	result, err := p.ParseFromFile(runFile)
	if err != nil {
		return err
	}

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			log.Println(e)
		}
		return ErrRun
	}

	var executorOpts []executor.Option

	// Enable logging
	shouldUseSimpleRenderer := verbose || os.Getenv("CI") == "true"
	var renderer echelon.LogRendered = renderers.NewSimpleRenderer(cmd.OutOrStdout(), nil)
	if !shouldUseSimpleRenderer {
		interactiveRenderer := renderers.NewInteractiveRenderer(cmd.OutOrStdout(), nil)
		go interactiveRenderer.StartDrawing()
		defer interactiveRenderer.StopDrawing()
		renderer = interactiveRenderer
	}
	logger := echelon.NewLogger(echelon.InfoLevel, renderer)
	if verbose {
		logger = echelon.NewLogger(echelon.DebugLevel, renderer)
	}
	executorOpts = append(executorOpts, executor.WithLogger(logger))

	// Enable a task filter if the task name is specified
	if len(args) == 1 {
		taskFilter := taskfilter.MatchExactTask(args[0])
		executorOpts = append(executorOpts, executor.WithTaskFilter(taskFilter))
	}

	// Dirty mode
	if dirty {
		executorOpts = append(executorOpts, executor.WithDirtyMode())
	}

	// Environment
	executorOpts = append(executorOpts, executor.WithEnvironment(envMap))

	// Run
	e, err := executor.New(filepath.Dir(runFile), result.Tasks, executorOpts...)
	if err != nil {
		return err
	}

	return e.Run(cmd.Context())
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [flags] [task]",
		Short: "Execute Cirrus CI tasks locally",
		RunE:  run,
		Args:  cobra.MaximumNArgs(1),
	}

	cmd.PersistentFlags().BoolVar(&dirty, "dirty", false, "if set the project directory will be mounted"+
		"in read-write mode, otherwise the project directory files are copied, taking .gitignore into account")
	cmd.PersistentFlags().StringArrayVarP(&environment, "environment", "e", []string{},
		"set (-e A=B) or pass-through (-e A) an environment variable")
	cmd.PersistentFlags().StringVarP(&runFile, "file", "f", ".cirrus.yml", "use file as the configuration file")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "")

	return cmd
}
