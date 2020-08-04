package commands

import (
	"errors"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	"github.com/cirruslabs/cirrus-cli/internal/executor/build/filter"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"log"
	"path/filepath"
)

var ErrRun = errors.New("run failed")

var runFile string
var verbose bool

func run(cmd *cobra.Command, args []string) error {
	// https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// Parse
	p := parser.Parser{}
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
	logger := logrus.New()
	logger.Out = cmd.OutOrStderr()
	if verbose {
		logger.Level = logrus.DebugLevel
	}
	executorOpts = append(executorOpts, executor.WithLogger(logger))

	// Configure a task filter based on the task pattern (if specified)
	if len(args) == 1 {
		taskFilter := filter.MatchTaskByPattern(args[0])
		executorOpts = append(executorOpts, executor.WithTaskFilter(taskFilter))
	}

	// Run
	e, err := executor.New(filepath.Dir(runFile), result.Tasks, executorOpts...)
	if err != nil {
		return err
	}

	return e.Run(cmd.Context())
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [flags] [task pattern]",
		Short: "Execute Cirrus CI tasks locally",
		RunE:  run,
		Args:  cobra.MaximumNArgs(1),
	}

	cmd.PersistentFlags().StringVarP(&runFile, "file", "f", ".cirrus.yml", "use file as the configuration file")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "")

	return cmd
}
