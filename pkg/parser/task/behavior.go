package task

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
)

type Behavior struct {
	commands []*api.Command

	parseable.DefaultParser
}

func NewBehavior() *Behavior {
	b := &Behavior{}

	scriptNameable := nameable.NewRegexNameable("(.*)script")
	b.OptionalField(scriptNameable, schema.TodoSchema, func(node *node.Node) error {
		command, err := handleScript(node, scriptNameable)
		if err != nil {
			return err
		}

		b.commands = append(b.commands, command)

		return nil
	})

	return b
}

func (b *Behavior) Parse(node *node.Node) error {
	if err := b.DefaultParser.Parse(node); err != nil {
		return err
	}

	return nil
}

func (b *Behavior) Proto() []*api.Command {
	return b.commands
}
