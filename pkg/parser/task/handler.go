package task

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
)

func handleBoolevatorField(node *node.Node, mergedEnv map[string]string) (bool, error) {
	onlyIfExpression, err := node.GetStringValue()
	if err != nil {
		return false, err
	}

	evaluation, err := boolevator.Eval(onlyIfExpression, mergedEnv, nil)
	if err != nil {
		return false, err
	}

	return evaluation, nil
}

func handleBackgroundScript(node *node.Node, nameable *nameable.RegexNameable) (*api.Command, error) {
	scripts, err := node.GetSliceOfNonEmptyStrings()
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
	scripts, err := node.GetSliceOfNonEmptyStrings()
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
