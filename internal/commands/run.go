//go:build linux || darwin || windows

package commands

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/chacha/pkg/localnetworkhelper"
	"github.com/cirruslabs/chacha/pkg/privdrop"
	"os"
	"runtime"
	"strings"
	"time"

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
)

const projectDir = "."

var ErrRun = errors.New("run failed")

// General flags.
var (
	artifactsDir                   string
	dirty                          bool
	heartbeatTimeoutRaw            string
	output                         string
	env                            []string
	envFile                        string
	affectedFiles                  []string
	affectedFilesGitRevision       string
	affectedFilesGitCachedRevision string
	verbose                        bool
)

// Common instance-related flags.
var lazyPull bool

// Container-related flags.
var (
	containerBackendType string
	containerLazyPull    bool
)

// Container-related flags: Dockerfile as CI environment[1] feature.
// [1]: https://cirrus-ci.org/guide/docker-builder-vm/#dockerfile-as-a-ci-environment
var (
	dockerfileImageTemplate string
	dockerfileImagePush     bool
)

// Tart-related flags.
var tartLazyPull bool

// Vetu-related flags.
var vetuLazyPull bool

// macOS-related flags.
var username string

// Flags useful for debugging.
var debugNoCleanup bool

func readYaml(
	ctx context.Context,
	baseEnvironment map[string]string,
	userSpecifiedEnvironment map[string]string,
) (*parser.Result, error) {
	// Retrieve the combined YAML configuration
	combinedYAML, err := helpers.ReadCombinedConfig(
		ctx,
		eenvironment.Merge(baseEnvironment, userSpecifiedEnvironment),
	)
	if err != nil {
		return nil, err
	}

	if affectedFilesGitRevision != "" {
		affectedFilesFromGit, err := helpers.GitDiff(projectDir, affectedFilesGitRevision, false)
		if err != nil {
			return nil, err
		}
		affectedFiles = append(affectedFiles, affectedFilesFromGit...)
	}

	if affectedFilesGitCachedRevision != "" {
		affectedFilesFromGit, err := helpers.GitDiff(projectDir, affectedFilesGitCachedRevision, true)
		if err != nil {
			return nil, err
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
	result, err := p.Parse(ctx, combinedYAML)
	if err != nil {
		if re, ok := err.(*parsererror.Rich); ok {
			fmt.Print(re.ContextLines())
		}

		return nil, err
	}

	return result, nil
}

func run(cmd *cobra.Command, args []string) error {
	var executorOpts []executor.Option

	// Run the macOS "Local Network" permission helper
	// when privilege dropping is requested
	if username != "" {
		localNetworkHelper, err := localnetworkhelper.New(cmd.Context())
		if err != nil {
			return err
		}

		executorOpts = append(executorOpts, executor.WithLocalNetworkHelper(localNetworkHelper))

		if err := privdrop.Drop(username); err != nil {
			return err
		}
	}

	baseEnvironment := makeBaseEnvironment()

	userSpecifiedEnvironment, err := makeUserSpecifiedEnvironment()
	if err != nil {
		return fmt.Errorf("%v: %v", ErrRun, err)
	}

	// https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	result, err := readYaml(cmd.Context(), baseEnvironment, userSpecifiedEnvironment)
	if err != nil {
		return err
	}

	// Enable logging
	logger, cancel := logs.GetLogger(output, verbose, cmd.OutOrStdout(), os.Stdout)
	defer cancel()
	executorOpts = append(executorOpts, executor.WithLogger(logger))

	// Enable a task filter if the task name is specified
	if len(args) == 1 {
		taskFilter := taskfilter.MatchExactTask(args[0])
		executorOpts = append(executorOpts, executor.WithTaskFilter(taskFilter))
	}

	// Artifacts directory
	if artifactsDir != "" {
		executorOpts = append(executorOpts, executor.WithArtifactsDir(artifactsDir))
	}

	// Dirty mode
	if dirty {
		executorOpts = append(executorOpts, executor.WithDirtyMode())
	}

	// Heartbeat timeout
	if heartbeatTimeoutRaw != "" {
		heartbeatTimeout, err := time.ParseDuration(heartbeatTimeoutRaw)
		if err != nil {
			return fmt.Errorf("%w: failed to parse --heartbeat-timeout value %q: %v",
				ErrRun, heartbeatTimeoutRaw, err)
		}

		executorOpts = append(executorOpts, executor.WithHeartbeatTimeout(heartbeatTimeout))
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

	// Vetu-related options
	executorOpts = append(executorOpts, executor.WithVetuOptions(options.VetuOptions{
		LazyPull: lazyPull || vetuLazyPull,
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
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			completions := []string{}

			baseEnvironment := makeBaseEnvironment()

			userSpecifiedEnvironment, err := makeUserSpecifiedEnvironment()
			if err != nil {
				return completions, cobra.ShellCompDirectiveError
			}

			result, err := readYaml(cmd.Context(), baseEnvironment, userSpecifiedEnvironment)
			if err != nil {
				return completions, cobra.ShellCompDirectiveError
			}

			for _, task := range result.Tasks {
				completions = append(completions, task.Name)
			}

			return completions, cobra.ShellCompDirectiveNoFileComp
		},
	}

	// General flags
	cmd.PersistentFlags().StringVar(&artifactsDir, "artifacts-dir", "",
		"directory in which to save the artifacts")
	cmd.PersistentFlags().BoolVar(&dirty, "dirty", false, "if set the project directory will "+
		"be mounted in read-write mode, otherwise the project directory files are copied, taking .gitignore "+
		"into account")
	cmd.PersistentFlags().StringArrayVarP(&env, "env", "e", []string{},
		"set (-e NAME=VALUE) or pass-through (-e NAME) an environment variable")
	cmd.PersistentFlags().StringVar(&envFile, "env-file", "",
		"set (NAME=VALUE on a separate line) or pass-through (NAME on a separate line) "+
			"environment variables from the specified file")
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
	cmd.PersistentFlags().StringVar(&heartbeatTimeoutRaw, "heartbeat-timeout", "",
		"duration after which the task will be canceled if no heartbeats were received from the agent "+
			"running as a part of that task (the agent sends a heartbeat every minute, so "+
			"--heartbeat-timeout=\"2m\" will effectively cancel the task after two missed heartbeats)")

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
		false, "whether to push the image produced by the Dockerfile as CI environment feature")

	// Tart-related flags
	cmd.PersistentFlags().BoolVar(&tartLazyPull, "tart-lazy-pull", false,
		"attempt to pull Tart VM images only if they are missing locally (helpful in case of registry rate limits)")

	// Vetu-related flags
	cmd.PersistentFlags().BoolVar(&vetuLazyPull, "vetu-lazy-pull", false,
		"attempt to pull Vetu VM images only if they are missing locally (helpful in case of registry rate limits)")

	// macOS-related flags
	if runtime.GOOS == "darwin" {
		cmd.Flags().StringVar(&username, "user", "", "username to drop privileges to "+
			"(\"Local Network\" permission workaround: requires starting \"cirrus run\" as \"root\", the privileges "+
			"will be then dropped to the specified user after starting the \"cirrus localnetworkhelper\" helper process)")
	}

	// Flags useful for debugging
	cmd.PersistentFlags().BoolVar(&debugNoCleanup, "debug-no-cleanup", false,
		"don't remove containers and volumes after execution")
	_ = cmd.PersistentFlags().MarkHidden("debug-no-cleanup")

	// Deprecated flags
	cmd.PersistentFlags().StringArrayVar(&env, "environment", []string{},
		"deprecated, please use --env instead")
	_ = cmd.PersistentFlags().MarkHidden("environment")

	return cmd
}

func makeBaseEnvironment() map[string]string {
	return eenvironment.Merge(
		eenvironment.Static(),
		eenvironment.BuildID(),
		eenvironment.ProjectSpecific(projectDir),
	)
}

func makeUserSpecifiedEnvironment() (map[string]string, error) {
	var result map[string]string
	var err error

	// --env-file
	if envFile != "" {
		result, err = helpers.EnvFileToMap(envFile)
		if err != nil {
			return nil, err
		}
	}

	// --env
	result = eenvironment.Merge(result, helpers.EnvArgsToMap(env))

	return result, nil
}
