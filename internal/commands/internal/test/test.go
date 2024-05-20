//go:build linux || darwin || windows
// +build linux darwin windows

package test

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bmatcuk/doublestar"
	"github.com/cirruslabs/cirrus-cli/internal/commands/helpers"
	"github.com/cirruslabs/cirrus-cli/internal/commands/logs"
	"github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/pkg/larker"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/local"
	"github.com/cirruslabs/echelon"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

var ErrTest = errors.New("test failed")

var update bool
var output string
var reportFilename string
var env map[string]string

type Comparison struct {
	FoundDifference bool
	Message         string
	RawDetails      string
	Path            string
}

type CirrusAnnotation struct {
	Level      string `json:"string"`
	Message    string `json:"message"`
	RawDetails string `json:"raw_details"`
	Path       string `json:"path"`
	StartLine  int64  `json:"start_line"`
	EndLine    int64  `json:"end_line"`
}

func (comparison *Comparison) AsCirrusAnnotation() *CirrusAnnotation {
	if !comparison.FoundDifference {
		return nil
	}

	return &CirrusAnnotation{
		Level:      "failure",
		Message:    comparison.Message,
		RawDetails: comparison.RawDetails,
		Path:       comparison.Path,
	}
}

// compareConfig compares generated configuration against an expected one.
func compareConfig(logger *echelon.Logger, testDir string, yamlConfig string) (*Comparison, error) {
	expectedConfigFilename := filepath.Join(testDir, ".cirrus.expected.yml")
	expectedConfigBytes, err := os.ReadFile(expectedConfigFilename)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTest, err)
	}

	comparison := logDifferenceIfAny(logger, "YAML", string(expectedConfigBytes), yamlConfig)
	comparison.Path = expectedConfigFilename

	if update && comparison.FoundDifference {
		if err := os.WriteFile(expectedConfigFilename, []byte(yamlConfig), 0600); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrTest, err)
		}
		comparison.FoundDifference = false
	}

	return comparison, nil
}

// compareLogs compares generated log against an expected one.
func compareLogs(logger *echelon.Logger, testDir string, actualLogs []byte) (*Comparison, error) {
	logsFilename := filepath.Join(testDir, ".cirrus.expected.log")
	logsBytes, err := os.ReadFile(logsFilename)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: %v", ErrTest, err)
	}

	comparison := logDifferenceIfAny(logger, "logs", string(logsBytes), string(actualLogs))
	comparison.Path = logsFilename

	if update && comparison.FoundDifference {
		if err := os.WriteFile(logsFilename, actualLogs, 0600); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrTest, err)
		}
		comparison.FoundDifference = false
	}

	return comparison, nil
}

func writeReport(yamlComparison, logsComparison *Comparison) error {
	if reportFilename == "" {
		return nil
	}

	var annotations []*CirrusAnnotation

	if annotation := yamlComparison.AsCirrusAnnotation(); annotation != nil {
		annotations = append(annotations, annotation)
	}
	if annotation := logsComparison.AsCirrusAnnotation(); annotation != nil {
		annotations = append(annotations, annotation)
	}

	if len(annotations) == 0 {
		return nil
	}

	reportFile, err := os.Create(reportFilename)
	if err != nil {
		return err
	}

	for _, annotation := range annotations {
		jsonBytes, err := json.Marshal(annotation)
		if err != nil {
			_ = reportFile.Close()
			_ = os.Remove(reportFilename)
			return err
		}

		if _, err := fmt.Fprintln(reportFile, string(jsonBytes)); err != nil {
			return err
		}
	}

	return reportFile.Close()
}

func test(cmd *cobra.Command, args []string) error {
	// https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// Discover tests
	var testDirs []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Does it look like a Starlark test?
		if info.Name() != ".cirrus.expected.yml" {
			return nil
		}

		ok, err := match(args, path)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		testDirs = append(testDirs, filepath.Dir(path))

		return nil
	})
	if err != nil {
		return err
	}

	if len(args) != 0 && len(testDirs) == 0 {
		return helpers.NewExitCodeError(2, fmt.Errorf("no tests matched"))
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
			larker.WithEnvironment(environment.Merge(testConfig.Environment, env)),
			larker.WithAffectedFiles(testConfig.AffectedFiles),
		)
		lrk := larker.New(larkerOpts...)

		sourceBytes, err := os.ReadFile(filepath.Join(testDir, ".cirrus.star"))
		if err != nil {
			return fmt.Errorf("%w: %v", ErrTest, err)
		}

		result, err := lrk.MainOptional(cmd.Context(), string(sourceBytes))
		if err != nil {
			return fmt.Errorf("%w: %v", ErrTest, err)
		}

		yamlComparison, err := compareConfig(logger, testDir, result.YAMLConfig)
		if err != nil {
			return err
		}
		logsComparison, err := compareLogs(logger, testDir, result.OutputLogs)
		if err != nil {
			return err
		}

		if err := writeReport(yamlComparison, logsComparison); err != nil {
			return err
		}

		// Should we consider the test as failed?
		if yamlComparison.FoundDifference || logsComparison.FoundDifference {
			someTestsFailed = true
		}

		logger.Finish(!yamlComparison.FoundDifference && !logsComparison.FoundDifference)
	}

	logger.Finish(!someTestsFailed)
	if someTestsFailed {
		return fmt.Errorf("%w: some tests failed", ErrTest)
	}

	return nil
}

func match(globs []string, path string) (bool, error) {
	if len(globs) == 0 {
		return true, nil
	}

	for _, glob := range globs {
		fullMatch, err := doublestar.PathMatch(glob, filepath.Dir(path))
		if err != nil {
			return false, err
		}
		if fullMatch {
			return true, nil
		}

		directoryMatch, err := doublestar.PathMatch(glob, filepath.Base(filepath.Dir(path)))
		if err != nil {
			return false, err
		}
		if directoryMatch {
			return true, nil
		}
	}

	return false, nil
}

func logDifferenceIfAny(logger *echelon.Logger, where string, a, b string) *Comparison {
	if a == b {
		return &Comparison{FoundDifference: false}
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(a, b, false)

	if len(diffs) == 0 {
		return &Comparison{FoundDifference: false}
	}

	logger.Warnf("Detected difference in %s:", where)
	logger.Warnf(dmp.DiffPrettyText(diffs))

	return &Comparison{
		FoundDifference: true,
		Message:         fmt.Sprintf("Actual result differs for %s", where),
		RawDetails:      b,
	}
}

func NewTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test [GLOB ...]",
		Short: "Discover and run Starlark tests",
		RunE:  test,
	}

	cmd.PersistentFlags().StringToStringVarP(&env, "env", "e", map[string]string{},
		"environment variable overrides to use")

	cmd.PersistentFlags().BoolVar(&update, "update", false,
		"update tests with differing .cirrus.expected.yml or .cirrus.expected.log, instead of failing them")

	cmd.PersistentFlags().StringVarP(&output, "output", "o", logs.DefaultFormat(), fmt.Sprintf("output format of logs, "+
		"supported values: %s", strings.Join(logs.Formats(), ", ")))

	cmd.PersistentFlags().StringVar(&reportFilename, "report", "",
		"additionally write a report in Cirrus Annotation Format (https://github.com/cirruslabs/cirrus-ci-annotations) "+
			"to this file")

	return cmd
}
