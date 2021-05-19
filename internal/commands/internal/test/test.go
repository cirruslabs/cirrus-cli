// +build linux darwin windows

package test

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/commands/logs"
	"github.com/cirruslabs/cirrus-cli/pkg/larker"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/local"
	"github.com/cirruslabs/echelon"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var ErrTest = errors.New("test failed")

var update bool
var output string

func test(cmd *cobra.Command, args []string) error {
	// https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// Discover tests
	var testDirs []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		// Does it look like a Starlark test?
		if info.Name() == ".cirrus.expected.yml" {
			testDirs = append(testDirs, filepath.Dir(path))
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Configure hierarchical progress renderer
	logger, cancel := logs.GetLogger(output, false, cmd.OutOrStdout(), os.Stdout)
	defer cancel()

	// Run tests
	var someTestsFailed bool

	for _, testDir := range testDirs {
		logger := logger.Scoped(testDir)

		// Create Starlark executor and run .cirrus.star to generate the configuration
		larkerOpts := []larker.Option{larker.WithTestMode()}

		fs := local.New(".")
		fs.Chdir(testDir)
		larkerOpts = append(larkerOpts, larker.WithFileSystem(fs))

		testConfig, err := LoadConfiguration(filepath.Join(testDir, ".cirrus.testconfig.yml"))
		if err != nil {
			return err
		}
		larkerOpts = append(larkerOpts,
			larker.WithEnvironment(testConfig.Environment),
			larker.WithAffectedFiles(testConfig.AffectedFiles),
		)

		lrk := larker.New(larkerOpts...)

		sourceBytes, err := ioutil.ReadFile(filepath.Join(testDir, ".cirrus.star"))
		if err != nil {
			return fmt.Errorf("%w: %v", ErrTest, err)
		}

		result, err := lrk.Main(cmd.Context(), string(sourceBytes))
		if err != nil {
			return fmt.Errorf("%w: %v", ErrTest, err)
		}

		// Compare generated configuration against an expected one
		expectedConfigFilename := filepath.Join(testDir, ".cirrus.expected.yml")
		expectedConfigBytes, err := ioutil.ReadFile(expectedConfigFilename)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrTest, err)
		}

		differentConfig := logDifferenceIfAny(logger, "YAML", string(expectedConfigBytes), result.YAMLConfig)

		if update && differentConfig {
			if err := ioutil.WriteFile(expectedConfigFilename, []byte(result.YAMLConfig), 0600); err != nil {
				return fmt.Errorf("%w: %v", ErrTest, err)
			}
			differentConfig = false
		}

		// Compare generated log against an expected one
		logsFilename := filepath.Join(testDir, ".cirrus.expected.log")
		logsBytes, err := ioutil.ReadFile(logsFilename)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %v", ErrTest, err)
		}
		differentLogs := logDifferenceIfAny(logger, "logs", string(logsBytes), string(result.OutputLogs))

		if update && differentLogs {
			if err := ioutil.WriteFile(logsFilename, result.OutputLogs, 0600); err != nil {
				return fmt.Errorf("%w: %v", ErrTest, err)
			}
			differentLogs = false
		}

		// Should we consider the test as failed?
		if differentConfig || differentLogs {
			someTestsFailed = true
		}

		logger.Finish(!differentConfig && !differentLogs)
	}

	logger.Finish(!someTestsFailed)
	if someTestsFailed {
		return fmt.Errorf("%w: some tests failed", ErrTest)
	}

	return nil
}

func logDifferenceIfAny(logger *echelon.Logger, where string, a, b string) bool {
	if a == b {
		return false
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(a, b, false)

	if len(diffs) == 0 {
		return false
	}

	logger.Warnf("Detected difference in %s:", where)
	logger.Warnf(dmp.DiffPrettyText(diffs))

	return true
}

func NewTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Discover and run Starlark tests",
		RunE:  test,
	}

	cmd.PersistentFlags().BoolVar(&update, "update", false,
		"update tests with differing .cirrus.expected.yml or .cirrus.expected.log, instead of failing them")

	cmd.PersistentFlags().StringVarP(&output, "output", "o", logs.DefaultFormat(), fmt.Sprintf("output format of logs, "+
		"supported values: %s", strings.Join(logs.Formats(), ", ")))

	return cmd
}
