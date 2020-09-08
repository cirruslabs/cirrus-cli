package commands

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/rpcparser"
	"github.com/spf13/cobra"
	"log"
)

var ErrValidate = errors.New("validate failed")

var validateFile string
var yaml bool

func validate(cmd *cobra.Command, args []string) error {
	// https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// Parse
	p := rpcparser.Parser{}
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

	if yaml {
		fmt.Println(string(testutil.TasksToJSON(nil, result.Tasks)))
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

	// A hidden flag to dump YAML representation of tasks and aid in generating test
	// cases for smooth rpcparser → parser transition
	cmd.PersistentFlags().BoolVar(&yaml, "json", false, "emit a JSON list with tasks contained in the configuration file")
	if err := cmd.PersistentFlags().MarkHidden("json"); err != nil {
		panic(err)
	}

	return cmd
}
