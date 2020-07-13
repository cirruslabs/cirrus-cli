package commands

import (
	"errors"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/spf13/cobra"
	"log"
)

var ErrValidate = errors.New("validation failed")

var file string

func validate(cmd *cobra.Command, args []string) error {
	// https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	p := parser.Parser{}
	result, err := p.ParseFromFile(file)
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

	cmd.PersistentFlags().StringVarP(&file, "file", "f", ".cirrus.yml", "use file as the configuration file")

	return cmd
}
