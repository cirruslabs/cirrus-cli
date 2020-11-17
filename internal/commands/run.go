// +build linux darwin windows

package commands

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/commands/logs"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	eenvironment "github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/cirruslabs/cirrus-cli/internal/executor/taskfilter"
	"github.com/cirruslabs/cirrus-cli/pkg/larker"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/local"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/cirruslabs/cirrus-cli/pkg/rpcparser"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var ErrRun = errors.New("run failed")

// General flags.
var dirty bool
var output string
var environment []string
var verbose bool

// Container-related flags.
var containerBackend string
var containerNoPull bool

// Flags useful for debugging.
var debugNoCleanup bool

// Experimental features flags.
var experimentalParser bool

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

func readYAMLConfig(path string) (string, error) {
	yamlConfig, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	return string(yamlConfig), nil
}

func readStarlarkConfig(ctx context.Context, path string, env map[string]string) (string, error) {
	starlarkSource, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	lrk := larker.New(larker.WithFileSystem(local.New(".")), larker.WithEnvironment(env))
	return lrk.Main(ctx, string(starlarkSource))
}

func readCombinedConfig(ctx context.Context, env map[string]string) (string, error) {
	yamlConfig, err := readYAMLConfig(".cirrus.yml")
	if err != nil {
		return "", err
	}

	starlarkConfig, err := readStarlarkConfig(ctx, ".cirrus.star", env)
	if err != nil {
		if os.IsNotExist(err) {
			return yamlConfig, nil
		}
		return "", err
	}

	return yamlConfig + "\n" + starlarkConfig, nil
}

func run(cmd *cobra.Command, args []string) error {
	// https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	backend, err := containerbackend.New(containerBackend)
	if err != nil {
		return err
	}

	projectDir := "."
	baseEnvironment := eenvironment.Merge(
		eenvironment.Static(),
		eenvironment.BuildID(),
		eenvironment.ProjectSpecific(projectDir),
	)
	userSpecifiedEnvironment := envArgsToMap(environment)

	// Retrieve the combined YAML configuration
	combinedYAML, err := readCombinedConfig(cmd.Context(), eenvironment.Merge(baseEnvironment, userSpecifiedEnvironment))
	if err != nil {
		return err
	}

	// Parse
	var result *parser.Result
	if experimentalParser {
		p := parser.New(parser.WithEnvironment(userSpecifiedEnvironment))
		result, err = p.Parse(cmd.Context(), combinedYAML)
		if err != nil {
			return err
		}
	} else {
		p := rpcparser.Parser{Environment: userSpecifiedEnvironment}
		r, err := p.Parse(combinedYAML)
		if err != nil {
			return err
		}

		// Convert into new parser result structure
		result = &parser.Result{
			Errors: r.Errors,
			Tasks:  r.Tasks,
		}
	}

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			log.Println(e)
		}
		return ErrRun
	}

	var executorOpts []executor.Option

	// Enable logging
	logger, cancel := logs.GetLogger(output, verbose, cmd.OutOrStdout(), os.Stdout)
	defer cancel()
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

	// Container-related options
	executorOpts = append(executorOpts, executor.WithContainerOptions(options.ContainerOptions{
		NoPull:    containerNoPull,
		NoCleanup: debugNoCleanup,
	}))

	// Environment
	executorOpts = append(executorOpts,
		executor.WithBaseEnvironmentOverride(baseEnvironment),
		executor.WithUserSpecifiedEnvironment(userSpecifiedEnvironment),
	)

	// Container backend
	executorOpts = append(executorOpts, executor.WithContainerBackend(backend))

	// Run
	e, err := executor.New(projectDir, result.Tasks, executorOpts...)
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

	// General flags
	cmd.PersistentFlags().BoolVar(&dirty, "dirty", false, "if set the project directory will be mounted"+
		"in read-write mode, otherwise the project directory files are copied, taking .gitignore into account")
	cmd.PersistentFlags().StringArrayVarP(&environment, "environment", "e", []string{},
		"set (-e A=B) or pass-through (-e A) an environment variable")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "")
	cmd.PersistentFlags().StringVarP(&output, "output", "o", logs.DefaultFormat(), fmt.Sprintf("output format of logs, "+
		"supported values: %s", strings.Join(logs.Formats(), ", ")))

	// Container-related flags
	cmd.PersistentFlags().StringVar(&containerBackend, "container-backend", containerbackend.BackendAuto,
		fmt.Sprintf("container engine backend to use, either \"%s\", \"%s\" or \"%s\"",
			containerbackend.BackendDocker, containerbackend.BackendPodman, containerbackend.BackendAuto))
	cmd.PersistentFlags().BoolVar(&containerNoPull, "container-no-pull", false,
		"don't attempt to pull the images before starting containers")

	// Deprecated flags
	cmd.PersistentFlags().BoolVar(&containerNoPull, "docker-no-pull", false,
		"don't attempt to pull the images before starting containers")
	_ = cmd.PersistentFlags().MarkDeprecated("docker-no-pull", "use --container-no-pull instead")

	// Flags useful for debugging
	cmd.PersistentFlags().BoolVar(&debugNoCleanup, "debug-no-cleanup", false,
		"don't remove containers and volumes after execution")

	// Experimental features flags
	cmd.PersistentFlags().BoolVar(&experimentalParser, "experimental-parser", false,
		"use local configuration parser instead of sending parse request to Cirrus Cloud")

	return cmd
}
