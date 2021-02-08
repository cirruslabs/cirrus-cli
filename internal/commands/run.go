// +build linux darwin windows

package commands

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/commands/helpers"
	"github.com/cirruslabs/cirrus-cli/internal/commands/logs"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	eenvironment "github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/cirruslabs/cirrus-cli/internal/executor/taskfilter"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/cirruslabs/cirrus-cli/pkg/rpcparser"
	"github.com/spf13/cobra"
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
var containerPull bool

// Container-related flags: Dockerfile as CI environment[1] feature.
// [1]: https://cirrus-ci.org/guide/docker-builder-vm/#dockerfile-as-a-ci-environment
var dockerfileImageTemplate string
var dockerfileImagePush bool

// Flags useful for debugging.
var debugNoCleanup bool

// Experimental features flags.
var experimentalParser bool

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
	userSpecifiedEnvironment := helpers.EnvArgsToMap(environment)

	// Retrieve the combined YAML configuration
	combinedYAML, err := helpers.ReadCombinedConfig(cmd.Context(),
		eenvironment.Merge(baseEnvironment, userSpecifiedEnvironment))
	if err != nil {
		return err
	}

	// Parse
	var tasks []*api.Task

	// nolint:nestif // this will be a no-issue once we switch to Go parser
	if experimentalParser {
		p := parser.New(parser.WithEnvironment(userSpecifiedEnvironment))
		result, err := p.Parse(cmd.Context(), combinedYAML)
		if err != nil {
			if re, ok := err.(*parsererror.Rich); ok {
				fmt.Print(re.ContextLines())
			}

			return err
		}
		tasks = result.Tasks
	} else {
		p := rpcparser.Parser{Environment: userSpecifiedEnvironment}
		result, err := p.Parse(combinedYAML)
		if err != nil {
			return err
		}
		tasks = result.Tasks
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
		Pull:      containerPull,
		NoCleanup: debugNoCleanup,

		DockerfileImageTemplate: dockerfileImageTemplate,
		DockerfileImagePush:     dockerfileImagePush,
	}))

	// Environment
	executorOpts = append(executorOpts,
		executor.WithBaseEnvironmentOverride(baseEnvironment),
		executor.WithUserSpecifiedEnvironment(userSpecifiedEnvironment),
	)

	// Container backend
	executorOpts = append(executorOpts, executor.WithContainerBackend(backend))

	// Run
	e, err := executor.New(projectDir, tasks, executorOpts...)
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
	cmd.PersistentFlags().BoolVar(&containerPull, "container-pull", false,
		"force pull the images before starting containers")

	// Container-related flags: Dockerfile as CI environment feature
	cmd.PersistentFlags().StringVar(&dockerfileImageTemplate, "dockerfile-image-template",
		"gcr.io/cirrus-ci-community/%s:latest", "image that Dockerfile as CI environment feature should produce")
	cmd.PersistentFlags().BoolVar(&dockerfileImagePush, "dockerfile-image-push",
		false, "whether to push whe image produced by the Dockerfile as CI environment feature")

	// Flags useful for debugging
	cmd.PersistentFlags().BoolVar(&debugNoCleanup, "debug-no-cleanup", false,
		"don't remove containers and volumes after execution")

	// Experimental features flags
	cmd.PersistentFlags().BoolVar(&experimentalParser, "experimental-parser", false,
		"use local configuration parser instead of sending parse request to Cirrus Cloud")

	return cmd
}
