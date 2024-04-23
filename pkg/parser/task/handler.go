package task

import (
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"strconv"
)

func handleBackgroundScript(node *node.Node, nameable *nameable.RegexNameable) (*api.Command, error) {
	scripts, err := node.GetScript()
	if err != nil {
		return nil, err
	}

	return &api.Command{
		Name: nameable.FirstGroupOrDefault(node.Name, "main"),
		Instruction: &api.Command_BackgroundScriptInstruction{
			BackgroundScriptInstruction: &api.BackgroundScriptInstruction{
				Scripts: scripts,
			},
		},
	}, nil
}

func handleScript(node *node.Node, nameable *nameable.RegexNameable) (*api.Command, error) {
	scripts, err := node.GetScript()
	if err != nil {
		return nil, err
	}

	return &api.Command{
		Name: nameable.FirstGroupOrDefault(node.Name, "main"),
		Instruction: &api.Command_ScriptInstruction{
			ScriptInstruction: &api.ScriptInstruction{
				Scripts: scripts,
			},
		},
	}, nil
}

func handleTimeoutIn(node *node.Node, mergedEnv map[string]string) (string, error) {
	timeoutHumanized, err := node.GetExpandedStringValue(mergedEnv)
	if err != nil {
		return "", err
	}

	// Convert "humanized" value to seconds
	timeoutSeconds, err := ParseSeconds(timeoutHumanized)
	if err != nil {
		return "", err
	}

	return strconv.FormatUint(uint64(timeoutSeconds), 10), nil
}
