package task

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
)

func handleOnlyIf(node *node.Node, mergedEnv map[string]string) (bool, error) {
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
