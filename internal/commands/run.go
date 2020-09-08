package commands

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/commands/logs"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	"github.com/cirruslabs/cirrus-cli/internal/executor/taskfilter"
	"github.com/cirruslabs/cirrus-cli/pkg/larker"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/rpcparser"
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var ErrRun = errors.New("run failed")

var dirty bool
var environment []string
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

func readYAMLConfig() (string, error) {
	yamlConfig, err := ioutil.ReadFile(".cirrus.yml")
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	return string(yamlConfig), nil
}

func readStarlarkConfig(ctx context.Context) (string, error) {
	starlarkSource, err := ioutil.ReadFile(".cirrus.star")
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	lrk := larker.New(larker.WithFileSystem(fs.NewLocalFileSystem(".")))
	return lrk.Main(ctx, string(starlarkSource))
}

func preflightCheck() error {
	// Since all of the instance types we currently support use Docker,
	// check that it's actually installed as early as possible
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("%w: cannot connect to Docker daemon: %v, make sure the Docker is installed",
			ErrRun, err)
	}
	defer cli.Close()

	return nil
}

func run(cmd *cobra.Command, args []string) error {
	// https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	if err := preflightCheck(); err != nil {
		return err
	}

	envMap := envArgsToMap(environment)

	// Retrieve configurations and merge them
	yamlConfig, err := readYAMLConfig()
	if err != nil {
		return err
	}

	starlarkConfig, err := readStarlarkConfig(cmd.Context())
	if err != nil {
		return err
	}

	mergedYAML := yamlConfig + "\n" + starlarkConfig

	// Parse
	p := rpcparser.Parser{Environment: envMap}
	result, err := p.Parse(mergedYAML)
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
	shouldUseSimpleRenderer := verbose || envVariableIsTrue("CI")
	var verboseRenderer = renderers.NewSimpleRenderer(cmd.OutOrStdout(), nil)
	var renderer echelon.LogRendered = verboseRenderer
	switch {
	case !shouldUseSimpleRenderer:
		interactiveRenderer := renderers.NewInteractiveRenderer(os.Stdout, nil)
		go interactiveRenderer.StartDrawing()
		defer interactiveRenderer.StopDrawing()
		renderer = interactiveRenderer
	case envVariableIsTrue("TRAVIS"):
		renderer = logs.NewTravisCILogsRenderer(verboseRenderer)
	case envVariableIsTrue("GITHUB_ACTIONS"):
		renderer = logs.NewGithubActionsLogsRenderer(verboseRenderer)
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
	e, err := executor.New(".", result.Tasks, executorOpts...)
	if err != nil {
		return err
	}

	return e.Run(cmd.Context())
}

func envVariableIsTrue(name string) bool {
	return os.Getenv(name) == "true"
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
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "")

	return cmd
}
