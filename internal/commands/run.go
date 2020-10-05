package commands

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/commands/logs"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	eenvironment "github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/cirruslabs/cirrus-cli/internal/executor/taskfilter"
	"github.com/cirruslabs/cirrus-cli/pkg/larker"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/local"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/cirruslabs/cirrus-cli/pkg/rpcparser"
	"github.com/docker/docker/client"
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

// Docker-related flags.
var dockerNoPull bool

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

func readStarlarkConfig(ctx context.Context, env map[string]string) (string, error) {
	starlarkSource, err := ioutil.ReadFile(".cirrus.star")
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	lrk := larker.New(larker.WithFileSystem(local.New(".")), larker.WithEnvironment(env))
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

	projectDir := "."
	baseEnvironment := eenvironment.Merge(
		eenvironment.Static(),
		eenvironment.BuildID(),
		eenvironment.ProjectSpecific(projectDir),
	)
	userSpecifiedEnvironment := envArgsToMap(environment)

	// Retrieve configurations and merge them
	yamlConfig, err := readYAMLConfig()
	if err != nil {
		return err
	}

	starlarkConfig, err := readStarlarkConfig(cmd.Context(), eenvironment.Merge(baseEnvironment, userSpecifiedEnvironment))
	if err != nil {
		return err
	}

	mergedYAML := yamlConfig + "\n" + starlarkConfig

	// Parse
	var result *parser.Result
	if experimentalParser {
		p := parser.New(parser.WithEnvironment(userSpecifiedEnvironment))
		result, err = p.Parse(cmd.Context(), mergedYAML)
		if err != nil {
			return err
		}
	} else {
		p := rpcparser.Parser{Environment: userSpecifiedEnvironment}
		r, err := p.Parse(mergedYAML)
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

	// Docker-related options
	executorOpts = append(executorOpts, executor.WithDockerOptions(options.DockerOptions{
		NoPull:    dockerNoPull,
		NoCleanup: debugNoCleanup,
	}))

	// Environment
	executorOpts = append(executorOpts,
		executor.WithBaseEnvironmentOverride(baseEnvironment),
		executor.WithUserSpecifiedEnvironment(userSpecifiedEnvironment),
	)

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

	// Docker-related flags
	cmd.PersistentFlags().BoolVar(&dockerNoPull, "docker-no-pull", false,
		"don't attempt to pull the images before starting containers")

	// Flags useful for debugging
	cmd.PersistentFlags().BoolVar(&debugNoCleanup, "debug-no-cleanup", false,
		"don't remove containers and volumes after execution")

	// Experimental features flags
	cmd.PersistentFlags().BoolVar(&experimentalParser, "experimental-parser", false,
		"use local configuration parser instead of sending parse request to Cirrus Cloud")

	return cmd
}
