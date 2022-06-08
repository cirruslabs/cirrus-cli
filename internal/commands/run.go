//go:build linux || darwin || windows
// +build linux darwin windows

package commands

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/commands/helpers"
	"github.com/cirruslabs/cirrus-cli/internal/commands/logs"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	eenvironment "github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/cirruslabs/cirrus-cli/internal/executor/taskfilter"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/local"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

var ErrRun = errors.New("run failed")

// General flags.
var dirty bool
var output string
var environment []string
var affectedFiles []string
var affectedFilesGitRevision string
var affectedFilesGitCachedRevision string
var verbose bool

// Common instance-related flags.
var lazyPull bool

// Container-related flags.
var containerBackendType string
var containerLazyPull bool

// Container-related flags: Dockerfile as CI environment[1] feature.
// [1]: https://cirrus-ci.org/guide/docker-builder-vm/#dockerfile-as-a-ci-environment
var dockerfileImageTemplate string
var dockerfileImagePush bool

// Tart-related flags.
var tartLazyPull bool

// Flags useful for debugging.
var debugNoCleanup bool

func run(cmd *cobra.Command, args []string) error {
	// https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

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

	if affectedFilesGitRevision != "" {
		affectedFilesFromGit, err := helpers.GitDiff(projectDir, affectedFilesGitRevision, false)
		if err != nil {
			return err
		}
		affectedFiles = append(affectedFiles, affectedFilesFromGit...)
	}

	if affectedFilesGitCachedRevision != "" {
		affectedFilesFromGit, err := helpers.GitDiff(projectDir, affectedFilesGitCachedRevision, true)
		if err != nil {
			return err
		}
		affectedFiles = append(affectedFiles, affectedFilesFromGit...)
	}

	// Parse
	p := parser.New(
		parser.WithEnvironment(eenvironment.Merge(eenvironment.Static(), userSpecifiedEnvironment)),
		parser.WithMissingInstancesAllowed(),
		parser.WithAffectedFiles(affectedFiles),
		parser.WithFileSystem(local.New(projectDir)),
	)
	result, err := p.Parse(cmd.Context(), combinedYAML)
	if err != nil {
		if re, ok := err.(*parsererror.Rich); ok {
			fmt.Print(re.ContextLines())
		}

		return err
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
		LazyPull:  lazyPull || containerLazyPull,
		NoCleanup: debugNoCleanup,

		DockerfileImageTemplate: dockerfileImageTemplate,
		DockerfileImagePush:     dockerfileImagePush,
	}))

	// Tart-related options
	executorOpts = append(executorOpts, executor.WithTartOptions(options.TartOptions{
		LazyPull: lazyPull || tartLazyPull,
	}))

	// Environment
	executorOpts = append(executorOpts,
		executor.WithBaseEnvironmentOverride(baseEnvironment),
		executor.WithUserSpecifiedEnvironment(userSpecifiedEnvironment),
	)

	// Container backend
	executorOpts = append(executorOpts, executor.WithContainerBackendType(containerBackendType))

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
	cmd.PersistentFlags().StringSliceVar(&affectedFiles, "affected-files", []string{},
		"comma-separated list of files to add to the list of affected files (used in changesInclude and "+
			"changesIncludeOnly functions)")
	cmd.PersistentFlags().StringVar(&affectedFilesGitRevision, "affected-files-git", "",
		"Git revision (e.g. HEAD, v0.1.0 or commit SHA) to compare unstaged changes against and "+
			"add changed files to the list of affected files (similarly to git diff)")
	cmd.PersistentFlags().StringVar(&affectedFilesGitCachedRevision, "affected-files-git-cached", "",
		"Git revision (e.g. HEAD, v0.1.0 or commit SHA) to compare staged changes against and "+
			"add changed files to the list of affected files (similarly to git diff --cached)")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "")
	cmd.PersistentFlags().StringVarP(&output, "output", "o", logs.DefaultFormat(), fmt.Sprintf("output format of logs, "+
		"supported values: %s", strings.Join(logs.Formats(), ", ")))

	// Common instance-related flags
	cmd.PersistentFlags().BoolVar(&lazyPull, "lazy-pull", false,
		"attempt to pull container and VM images only if they are missing locally "+
			"(helpful in case of registry rate limits; enables --container-lazy-pull and --tart-lazy-pull)")

	// Container-related flags
	cmd.PersistentFlags().StringVar(&containerBackendType, "container-backend", containerbackend.BackendTypeAuto,
		fmt.Sprintf("container engine backend to use, either \"%s\", \"%s\" or \"%s\"",
			containerbackend.BackendTypeDocker, containerbackend.BackendTypePodman, containerbackend.BackendTypeAuto))
	cmd.PersistentFlags().BoolVar(&containerLazyPull, "container-lazy-pull", false,
		"attempt to pull images only if they are missing locally (helpful in case of registry rate limits)")

	// Container-related flags: Dockerfile as CI environment feature
	cmd.PersistentFlags().StringVar(&dockerfileImageTemplate, "dockerfile-image-template",
		"gcr.io/cirrus-ci-community/%s:latest", "image that Dockerfile as CI environment feature should produce")
	cmd.PersistentFlags().BoolVar(&dockerfileImagePush, "dockerfile-image-push",
		false, "whether to push whe image produced by the Dockerfile as CI environment feature")

	// Tart-related flags
	cmd.PersistentFlags().BoolVar(&tartLazyPull, "tart-lazy-pull", false,
		"attempt to pull Tart VM images only if they are missing locally (helpful in case of registry rate limits)")

	// Flags useful for debugging
	cmd.PersistentFlags().BoolVar(&debugNoCleanup, "debug-no-cleanup", false,
		"don't remove containers and volumes after execution")
	_ = cmd.PersistentFlags().MarkHidden("debug-no-cleanup")

	return cmd
}
