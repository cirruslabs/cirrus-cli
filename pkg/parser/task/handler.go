package task

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
)

func handleOnlyIf(node *node.Node, env map[string]string) (bool, error) {
	onlyIfExpression, err := node.GetStringValue()
	if err != nil {
		return false, err
	}

	evaluation, err := boolevator.Eval(onlyIfExpression, env, nil)
	if err != nil {
		return false, err
	}

	return evaluation, nil
}
