package helpers

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/larker"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/local"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"strings"
)

var ErrConfigurationReadFailed = errors.New("failed to read configuration")

func ConsumeSubCommands(cmd *cobra.Command, subCommands []*cobra.Command) *cobra.Command {
	var hasValidSubcommands bool

	for _, subCommand := range subCommands {
		if cmd == nil {
			continue
		}

		cmd.AddCommand(subCommand)
		hasValidSubcommands = true
	}

	if hasValidSubcommands {
		return cmd
	}

	return nil
}

// envArgsToMap parses and expands environment arguments like "A=B" (set operation)
// and "A" (pass-through operation) into a map suitable for use across the codebase.
func EnvArgsToMap(arguments []string) map[string]string {
	result := make(map[string]string)

	const (
		keyPart = iota
		valuePart
		maxParts
	)

	for _, argument := range arguments {
		parts := strings.SplitN(argument, "=", maxParts)

		if len(parts) == maxParts {
			// Set mode: simply assign the provided value to key
			result[parts[keyPart]] = parts[valuePart]
		} else {
			// Pass-through mode: resolve the value for the provided key and assign it (if any)
			resolvedValue, found := os.LookupEnv(parts[keyPart])
			if !found {
				break
			}
			result[parts[keyPart]] = resolvedValue
		}
	}

	return result
}

func ReadYAMLConfig(path string) (string, error) {
	yamlConfig, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(yamlConfig), nil
}

func EvaluateStarlarkConfig(ctx context.Context, path string, env map[string]string) (string, error) {
	starlarkSource, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	lrk := larker.New(larker.WithFileSystem(local.New(".")), larker.WithEnvironment(env))
	return lrk.Main(ctx, string(starlarkSource))
}

func ReadCombinedConfig(ctx context.Context, env map[string]string) (string, error) {
	// Here we read the .cirrus.yaml first so that if the error would arise
	// and will be inspected it would indicate the preferable extension
	yamlConfig, yamlErr := ReadYAMLConfig(".cirrus.yaml")
	if yamlErr != nil {
		if !os.IsNotExist(yamlErr) {
			return "", yamlErr
		}

		yamlConfig, yamlErr = ReadYAMLConfig(".cirrus.yml")
		if yamlErr != nil && !os.IsNotExist(yamlErr) {
			return "", yamlErr
		}
	}

	starlarkConfig, starlarkErr := EvaluateStarlarkConfig(ctx, ".cirrus.star", env)
	if starlarkErr != nil && !os.IsNotExist(starlarkErr) {
		return "", starlarkErr
	}

	switch {
	case yamlErr == nil && starlarkErr == nil:
		return yamlConfig + "\n" + starlarkConfig, nil
	case yamlErr == nil:
		return yamlConfig, nil
	case starlarkErr == nil:
		return starlarkConfig, nil
	default:
		return "", fmt.Errorf("%w: neither .cirrus.yml (%s) nor .cirrus.star were accessible (%s)",
			ErrConfigurationReadFailed, yamlErr, starlarkErr)
	}
}
