package validate

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/commands/helpers"
	eenvironment "github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/cirruslabs/cirrus-cli/pkg/rpcparser"
	"github.com/spf13/cobra"
	"log"
	"strings"
)

var ErrValidate = errors.New("validate failed")

// General flags.
var validateFile string
var environment []string

// Experimental features flags.
var experimentalParser bool

var yaml bool

func validate(cmd *cobra.Command, args []string) error {
	// https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// Craft the environment
	baseEnvironment := eenvironment.Merge(
		eenvironment.Static(),
		eenvironment.BuildID(),
		eenvironment.ProjectSpecific("."),
	)
	userSpecifiedEnvironment := helpers.EnvArgsToMap(environment)
	resultingEnvironment := eenvironment.Merge(baseEnvironment, userSpecifiedEnvironment)

	// Retrieve a combined YAML configuration or a specific one if asked to
	var configuration string
	var err error

	switch {
	case validateFile == "":
		configuration, err = helpers.ReadCombinedConfig(cmd.Context(), resultingEnvironment)
		if err != nil {
			return err
		}
	case strings.HasSuffix(validateFile, ".yml"):
		configuration, err = helpers.ReadYAMLConfig(validateFile)
		if err != nil {
			return err
		}
	case strings.HasSuffix(validateFile, ".star"):
		configuration, err = helpers.EvaluateStarlarkConfig(cmd.Context(), validateFile, resultingEnvironment)
		if err != nil {
			return err
		}
	default:
		return ErrValidate
	}

	// Parse
	var result *parser.Result

	if experimentalParser {
		p := parser.New(parser.WithEnvironment(userSpecifiedEnvironment))
		result, err = p.Parse(cmd.Context(), configuration)
		if err != nil {
			return err
		}
	} else {
		p := rpcparser.Parser{Environment: userSpecifiedEnvironment}
		r, err := p.Parse(configuration)
		if err != nil {
			return err
		}

		// Convert into new parser result structure
		result = &parser.Result{
			Errors: r.Errors,
			Tasks:  r.Tasks,
		}
	}

	// Check for errors
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			log.Println(e)
		}
		return ErrValidate
	}

	if yaml {
		fmt.Println(string(testutil.TasksToJSON(nil, result.Tasks)))
	}

	return nil
}

func NewValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate Cirrus CI configuration file",
		RunE:  validate,
	}

	// General flags
	cmd.PersistentFlags().StringArrayVarP(&environment, "environment", "e", []string{},
		"set (-e A=B) or pass-through (-e A) an environment variable to the Starlark interpreter")
	cmd.PersistentFlags().StringVarP(&validateFile, "file", "f", "",
		"use file as the configuration file (the path should end with either .yml or ..star)")

	// Experimental features flags
	cmd.PersistentFlags().BoolVar(&experimentalParser, "experimental-parser", false,
		"use local configuration parser instead of sending parse request to Cirrus Cloud")

	// A hidden flag to dump YAML representation of tasks and aid in generating test
	// cases for smooth rpcparser → parser transition
	cmd.PersistentFlags().BoolVar(&yaml, "json", false, "emit a JSON list with tasks contained in the configuration file")
	if err := cmd.PersistentFlags().MarkHidden("json"); err != nil {
		panic(err)
	}

	return cmd
}
