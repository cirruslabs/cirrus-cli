package worker

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/endpoint"
	"github.com/cirruslabs/cirrus-cli/internal/worker"
	"github.com/cirruslabs/cirrus-cli/internal/worker/upstream"
	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"strings"
)

var ErrConfiguration = errors.New("configuration error")

type Config struct {
	Name  string `yaml:"name"`
	Token string `yaml:"token"`

	Labels    map[string]string  `yaml:"labels"`
	Resources map[string]float64 `yaml:"resources"`

	Log ConfigLog `yaml:"log"`

	RPC ConfigRPC `yaml:"rpc"`

	Upstreams []ConfigUpstream `yaml:"upstreams"`
}

type ConfigLog struct {
	Level        string `yaml:"level"`
	File         string `yaml:"file"`
	RotateSize   string `yaml:"rotate-size"`
	MaxRotations uint   `yaml:"max-rotations"`
}

type ConfigRPC struct {
	Endpoint string `yaml:"endpoint"`
}

type ConfigUpstream struct {
	Token    string `yaml:"token"`
	Endpoint string `yaml:"endpoint"`
}

var (
	configPath  string
	name        string
	token       string
	labels      map[string]string
	rpcEndpoint string
)

func attachFlags(cmd *cobra.Command) {
	// Default worker name to host name and make
	// it pretty by removing common suffixes
	defaultName, _ := os.Hostname()
	defaultName = strings.TrimSuffix(defaultName, ".lan")
	defaultName = strings.TrimSuffix(defaultName, ".local")

	cmd.PersistentFlags().StringVarP(&configPath, "file", "f", "",
		"configuration file path (e.g. /etc/cirrus/worker.yml)")
	cmd.PersistentFlags().StringVar(&name, "name", defaultName,
		"worker name to use when registering in the pool")
	cmd.PersistentFlags().StringVar(&token, "token", "", "pool registration token")
	cmd.PersistentFlags().StringToStringVar(&labels, "labels", map[string]string{},
		"additional labels to use (e.g. --labels distro=debian)")
	cmd.PersistentFlags().StringVar(&rpcEndpoint, "rpc-endpoint", upstream.DefaultRPCEndpoint,
		"RPC endpoint address")
}

func buildWorker(cmd *cobra.Command) (*worker.Worker, error) {
	// Instantiate a default configuration
	config := Config{
		Name:   name,
		Token:  token,
		Labels: labels,
		RPC: ConfigRPC{
			Endpoint: rpcEndpoint,
		},
	}

	// Load the YAML configuration file
	if configPath != "" {
		configBytes, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}

		if err := yaml.Unmarshal(configBytes, &config); err != nil {
			return nil, err
		}
	}

	// Configure worker
	opts := []worker.Option{
		worker.WithLabels(config.Labels),
		worker.WithResources(config.Resources),
	}

	// Configure logging
	logger := logrus.New()

	if config.Log.Level != "" {
		level, err := logrus.ParseLevel(config.Log.Level)
		if err != nil {
			return nil, err
		}

		logger.SetLevel(level)
	}

	var err error
	var output io.Writer

	if config.Log.File != "" {
		logRotateSizeBytes := uint64(0)
		if config.Log.RotateSize != "" {
			logRotateSizeBytes, err = humanize.ParseBytes(config.Log.RotateSize)
			if err != nil {
				return nil, fmt.Errorf("failed to parse log size for rotation: %w", err)
			}
		}

		output = &lumberjack.Logger{
			Filename:   config.Log.File,
			MaxSize:    int(logRotateSizeBytes / humanize.MByte),
			MaxBackups: int(config.Log.MaxRotations),
		}
	} else {
		output = cmd.ErrOrStderr()
	}
	logger.SetOutput(output)

	opts = append(opts, worker.WithLogger(logger))

	// Configure upstreams
	if len(config.Upstreams) == 0 {
		config.Upstreams = append(config.Upstreams, ConfigUpstream{
			Token:    config.Token,
			Endpoint: config.RPC.Endpoint,
		})
	} else {
		if config.Token != "" {
			return nil, fmt.Errorf("%w: \"token:\" and \"endpoints:\" are mutually exclusive",
				ErrConfiguration)
		}

		if config.RPC.Endpoint != upstream.DefaultRPCEndpoint {
			return nil, fmt.Errorf("%w: \"rpc:\" and \"endpoints:\" are mutually exclusive",
				ErrConfiguration)
		}
	}

	for _, configUpstream := range config.Upstreams {
		upstreamOpts := []upstream.Option{upstream.WithLogger(logger)}

		if configUpstream.Endpoint != "" {
			upstreamOpts = append(upstreamOpts, upstream.WithRPCEndpoint(configUpstream.Endpoint))
			upstreamOpts = append(upstreamOpts, upstream.WithAgentEndpoint(
				endpoint.NewRemote(configUpstream.Endpoint),
			))
		}

		upstream, err := upstream.New(config.Name, configUpstream.Token, upstreamOpts...)
		if err != nil {
			return nil, err
		}

		opts = append(opts, worker.WithUpstream(upstream))
	}

	// Instantiate worker
	return worker.New(opts...)
}
