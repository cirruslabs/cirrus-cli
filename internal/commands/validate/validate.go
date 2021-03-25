package validate

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/commands/helpers"
	"github.com/cirruslabs/cirrus-cli/internal/evaluator"
	eenvironment "github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/pkg/executorservice"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/spf13/cobra"
	"io"
	"strings"
)

var ErrValidate = errors.New("validate failed")

// General flags.
var validateFile string
var environment []string

func additionalInstancesOption(stderr io.Writer) parser.Option {
	// Try to retrieve additional instances from the Cirrus Cloud
	additionalInstances, err := executorservice.New().SupportedInstances()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "failed to retrieve additional instances supported by the Cirrus Cloud,"+
			"their validation will not be performed")

		return parser.WithMissingInstancesAllowed()
	}

	transformedInstances, err := evaluator.TransformAdditionalInstances(additionalInstances)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "failed to parse additional instances from the Cirrus Cloud,"+
			"their validation will not be performed")

		return parser.WithMissingInstancesAllowed()
	}

	return parser.WithAdditionalInstances(transformedInstances)
}

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
	case strings.HasSuffix(validateFile, ".yml") || strings.HasSuffix(validateFile, ".yaml"):
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
	p := parser.New(parser.WithEnvironment(userSpecifiedEnvironment), additionalInstancesOption(cmd.ErrOrStderr()))
	_, err = p.Parse(cmd.Context(), configuration)
	if err != nil {
		if re, ok := err.(*parsererror.Rich); ok {
			fmt.Print(re.ContextLines())
		}

		return err
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

	return cmd
}
