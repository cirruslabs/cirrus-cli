package worker

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/worker"
	"github.com/spf13/cobra"
)

var ErrRun = errors.New("run failed")

var (
	name   string
	token  string
	labels map[string]string
)

func run(cmd *cobra.Command, args []string) error {
	worker, err := worker.New(
		worker.WithName(name),
		worker.WithRegistrationToken(token),
		worker.WithLabels(labels),
	)
	if err != nil {
		return err
	}

	if err := worker.Run(cmd.Context()); err != nil {
		return fmt.Errorf("%w: %v", ErrRun, err)
	}

	return nil
}

func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run persistent worker",
		RunE:  run,
	}

	cmd.PersistentFlags().StringVar(&name, "name", "${hostname}-${n}", "worker name to use when registering in the pool")
	cmd.PersistentFlags().StringVar(&token, "token", "", "pool registration token")
	cmd.PersistentFlags().StringToStringVar(&labels, "labels", map[string]string{},
		"additional labels to use (e.g. --labels distro=debian)")

	return cmd
}
