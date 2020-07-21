package commands

import (
	"errors"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/spf13/cobra"
	"log"
)

var ErrValidate = errors.New("validate failed")

var validateFile string

func validate(cmd *cobra.Command, args []string) error {
	// https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// Parse
	p := parser.Parser{}
	result, err := p.ParseFromFile(validateFile)
	if err != nil {
		return err
	}

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			log.Println(e)
		}
		return ErrValidate
	}

	return nil
}

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate Cirrus CI configuration file",
		RunE:  validate,
	}

	cmd.PersistentFlags().StringVarP(&validateFile, "file", "f", ".cirrus.yml", "use file as the configuration file")

	return cmd
}
