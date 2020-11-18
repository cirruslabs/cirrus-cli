package helpers

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/larker"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/local"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"strings"
)

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
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	return string(yamlConfig), nil
}

func ReadStarlarkConfig(ctx context.Context, path string, env map[string]string) (string, error) {
	starlarkSource, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	lrk := larker.New(larker.WithFileSystem(local.New(".")), larker.WithEnvironment(env))
	return lrk.Main(ctx, string(starlarkSource))
}

func ReadCombinedConfig(ctx context.Context, env map[string]string) (string, error) {
	yamlConfig, err := ReadYAMLConfig(".cirrus.yml")
	if err != nil {
		return "", err
	}

	starlarkConfig, err := ReadStarlarkConfig(ctx, ".cirrus.star", env)
	if err != nil {
		if os.IsNotExist(err) {
			return yamlConfig, nil
		}
		return "", err
	}

	return yamlConfig + "\n" + starlarkConfig, nil
}
