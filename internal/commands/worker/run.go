package worker

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/worker"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
)

var ErrRun = errors.New("run failed")

var (
	configPath string
	name       string
	token      string
	labels     map[string]string

	// RPC-related variables.
	rpcEndpointAddress string
)

func run(cmd *cobra.Command, args []string) error {
	if configPath != "" {
		viper.SetConfigType("yaml")
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			return err
		}
	}

	opts := []worker.Option{
		worker.WithName(viper.GetString("name")),
		worker.WithRegistrationToken(viper.GetString("token")),
		worker.WithLabels(viper.GetStringMapString("labels")),
	}

	if rpcEndpointAddress != "" {
		opts = append(opts, worker.WithRPCEndpoint(rpcEndpointAddress))
	}

	worker, err := worker.New(opts...)
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

	cmd.PersistentFlags().StringVarP(&configPath, "file", "f", "", "configuration file path (e.g. /etc/cirrus/worker.yml)")

	// Default worker name to host name
	defaultName, _ := os.Hostname()
	// Make the default name pretty by removing common suffixes
	defaultName = strings.TrimSuffix(defaultName, ".lan")
	defaultName = strings.TrimSuffix(defaultName, ".local")
	cmd.PersistentFlags().StringVar(&name, "name", defaultName, "worker name to use when registering in the pool")
	_ = viper.BindPFlag("name", cmd.PersistentFlags().Lookup("name"))

	cmd.PersistentFlags().StringVar(&token, "token", "", "pool registration token")
	_ = viper.BindPFlag("token", cmd.PersistentFlags().Lookup("token"))

	cmd.PersistentFlags().StringToStringVar(&labels, "labels", map[string]string{},
		"additional labels to use (e.g. --labels distro=debian)")
	_ = viper.BindPFlag("labels", cmd.PersistentFlags().Lookup("labels"))

	// RPC-related variables
	cmd.PersistentFlags().StringVar(&rpcEndpointAddress, "rpc-endpoint", worker.DefaultRPCEndpoint, "RPC endpoint address")
	_ = viper.BindPFlag("rpc.endpoint", cmd.PersistentFlags().Lookup("rpc-endpoint"))
	_ = cmd.PersistentFlags().MarkHidden("rpc-endpoint")

	return cmd
}
