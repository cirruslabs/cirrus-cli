package task

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/task/command"
)

type Behavior struct {
	commands []*api.Command

	parseable.DefaultParser
}

func NewBehavior(mergedEnv map[string]string, boolevator *boolevator.Boolevator) *Behavior {
	b := &Behavior{}

	bgNameable := nameable.NewRegexNameable("^(.*)background_script$")
	b.OptionalField(bgNameable, schema.TodoSchema, func(node *node.Node) error {
		command, err := handleBackgroundScript(node, bgNameable)
		if err != nil {
			return err
		}

		b.commands = append(b.commands, command)

		return nil
	})

	scriptNameable := nameable.NewRegexNameable("^(.*)script$")
	b.OptionalField(scriptNameable, schema.TodoSchema, func(node *node.Node) error {
		command, err := handleScript(node, scriptNameable)
		if err != nil {
			return err
		}

		b.commands = append(b.commands, command)

		return nil
	})

	cacheNameable := nameable.NewRegexNameable("^(.*)cache$")
	b.OptionalField(cacheNameable, schema.TodoSchema, func(node *node.Node) error {
		cache := NewCacheCommand(mergedEnv, boolevator)
		if err := cache.Parse(node); err != nil {
			return err
		}
		b.commands = append(b.commands, cache.Proto())
		return nil
	})

	artifactsNameable := nameable.NewRegexNameable("^(.*)artifacts$")
	b.OptionalField(artifactsNameable, schema.TodoSchema, func(node *node.Node) error {
		artifacts := command.NewArtifactsCommand(mergedEnv)
		if err := artifacts.Parse(node); err != nil {
			return err
		}
		b.commands = append(b.commands, artifacts.Proto())
		return nil
	})

	fileNameable := nameable.NewRegexNameable("^(.*)file$")
	b.OptionalField(fileNameable, schema.TodoSchema, func(node *node.Node) error {
		file := command.NewFileCommand(mergedEnv)
		if err := file.Parse(node); err != nil {
			return err
		}
		b.commands = append(b.commands, file.Proto())
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
