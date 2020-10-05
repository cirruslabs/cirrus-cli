package internal

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/commands/logs"
	"github.com/cirruslabs/cirrus-cli/pkg/larker"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/local"
	"github.com/go-test/deep"
	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var ErrTest = errors.New("test failed")

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

		// Load expected configuration
		expectedConfigBytes, err := ioutil.ReadFile(filepath.Join(testDir, ".cirrus.expected.yml"))
		if err != nil {
			return fmt.Errorf("%w: %v", ErrTest, err)
		}

		// Create Starlark executor and run .cirrus.star to generate the configuration
		fs := local.New(testDir)
		lrk := larker.New(larker.WithFileSystem(fs))

		sourceBytes, err := ioutil.ReadFile(filepath.Join(testDir, ".cirrus.star"))
		if err != nil {
			return fmt.Errorf("%w: %v", ErrTest, err)
		}

		generatedConfigString, err := lrk.Main(cmd.Context(), string(sourceBytes))
		if err != nil {
			return fmt.Errorf("%w: %v", ErrTest, err)
		}

		// Compare generated configuration with the expected configuration
		var expectedConfig yaml.MapSlice
		err = yaml.UnmarshalWithOptions(expectedConfigBytes, &expectedConfig, yaml.UseOrderedMap())
		if err != nil {
			return fmt.Errorf("%w: %v", ErrTest, err)
		}

		var generatedConfig yaml.MapSlice
		err = yaml.UnmarshalWithOptions([]byte(generatedConfigString), &generatedConfig, yaml.UseOrderedMap())
		if err != nil {
			return fmt.Errorf("%w: %v", ErrTest, err)
		}

		diff := deep.Equal(expectedConfig, generatedConfig)
		currentTestSucceeded := len(diff) == 0

		if !currentTestSucceeded {
			for _, line := range diff {
				logger.Warnf("%s", line)
			}
			someTestsFailed = true
		}

		logger.Finish(currentTestSucceeded)
	}

	logger.Finish(someTestsFailed)
	if someTestsFailed {
		return fmt.Errorf("%w: some tests failed", ErrTest)
	}

	return nil
}

func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Discover and run Starlark tests",
		RunE:  test,
	}

	cmd.PersistentFlags().StringVarP(&output, "output", "o", logs.DefaultFormat(), fmt.Sprintf("output format of logs, "+
		"supported values: %s", strings.Join(logs.Formats(), ", ")))

	return cmd
}
