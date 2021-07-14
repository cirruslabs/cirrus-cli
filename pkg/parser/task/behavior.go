package task

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/task/command"
	jsschema "github.com/lestrrat-go/jsschema"
)

type Behavior struct {
	commands []*api.Command

	parseable.DefaultParser
}

func NewBehavior(
	mergedEnv map[string]string,
	boolevator *boolevator.Boolevator,
	previousCommands []*api.Command,
) *Behavior {
	b := &Behavior{}

	bgNameable := nameable.NewRegexNameable("^(.*)background_script$")
	b.OptionalField(bgNameable, schema.Script(""), func(node *node.Node) error {
		command, err := handleBackgroundScript(node, bgNameable)
		if err != nil {
			return err
		}

		b.commands = append(b.commands, command)

		return nil
	})

	scriptNameable := nameable.NewRegexNameable("^(.*)script$")
	b.OptionalField(scriptNameable, schema.Script(""), func(node *node.Node) error {
		command, err := handleScript(node, scriptNameable)
		if err != nil {
			return err
		}

		b.commands = append(b.commands, command)

		return nil
	})

	cacheNameable := nameable.NewRegexNameable("^(.*)cache$")
	cacheSchema := command.NewCacheCommand(nil, nil).Schema()
	b.OptionalField(cacheNameable, cacheSchema, func(node *node.Node) error {
		cache := command.NewCacheCommand(mergedEnv, boolevator)
		if err := cache.Parse(node); err != nil {
			return err
		}
		b.commands = append(b.commands, cache.Proto())
		return nil
	})

	uploadCachesNameable := nameable.NewSimpleNameable("upload_caches")
	b.OptionalField(uploadCachesNameable, command.UploadCachesSchema(), func(node *node.Node) error {
		commandsToAppend, err := command.UploadCachesHelper(mergedEnv, append(previousCommands, b.commands...), node)
		if err != nil {
			return err
		}

		b.commands = append(b.commands, commandsToAppend...)

		return nil
	})

	artifactsNameable := nameable.NewRegexNameable("^(.*)artifacts$")
	artifactsSchema := command.NewArtifactsCommand(nil).Schema()
	b.OptionalField(artifactsNameable, artifactsSchema, func(node *node.Node) error {
		artifacts := command.NewArtifactsCommand(mergedEnv)
		if err := artifacts.Parse(node); err != nil {
			return err
		}
		b.commands = append(b.commands, artifacts.Proto())
		return nil
	})

	fileNameable := nameable.NewRegexNameable("^(.*)file$")
	fileSchema := command.NewFileCommand(nil).Schema()
	b.OptionalField(fileNameable, fileSchema, func(node *node.Node) error {
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
	return b.DefaultParser.Parse(node)
}

func (b *Behavior) Proto() []*api.Command {
	return b.commands
}

func (b *Behavior) Schema() *jsschema.Schema {
	modifiedSchema := b.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}

	return modifiedSchema
}
