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

// compareConfig compares generated configuration against an expected one.
func compareConfig(logger *echelon.Logger, testDir string, yamlConfig string) (bool, error) {
	expectedConfigFilename := filepath.Join(testDir, ".cirrus.expected.yml")
	expectedConfigBytes, err := ioutil.ReadFile(expectedConfigFilename)
	if err != nil {
		return true, fmt.Errorf("%w: %v", ErrTest, err)
	}

	differentConfig := logDifferenceIfAny(logger, "YAML", string(expectedConfigBytes), yamlConfig)

	if update && differentConfig {
		if err := ioutil.WriteFile(expectedConfigFilename, []byte(yamlConfig), 0600); err != nil {
			return true, fmt.Errorf("%w: %v", ErrTest, err)
		}
		differentConfig = false
	}

	return differentConfig, nil
}

// compareLogs compares generated log against an expected one.
func compareLogs(logger *echelon.Logger, testDir string, actualLogs []byte) (bool, error) {
	logsFilename := filepath.Join(testDir, ".cirrus.expected.log")
	logsBytes, err := ioutil.ReadFile(logsFilename)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return true, fmt.Errorf("%w: %v", ErrTest, err)
	}

	differentLogs := logDifferenceIfAny(logger, "logs", string(logsBytes), string(actualLogs))

	if update && differentLogs {
		if err := ioutil.WriteFile(logsFilename, actualLogs, 0600); err != nil {
			return true, fmt.Errorf("%w: %v", ErrTest, err)
		}
		differentLogs = false
	}

	return differentLogs, nil
}

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
		if err != nil && !errors.Is(err, larker.ErrNotFound) {
			return fmt.Errorf("%w: %v", ErrTest, err)
		}

		differentConfig, err := compareConfig(logger, testDir, result.YAMLConfig)
		if err != nil {
			return err
		}
		differentLogs, err := compareLogs(logger, testDir, result.OutputLogs)
		if err != nil {
			return err
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
