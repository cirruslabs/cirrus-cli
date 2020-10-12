package task

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"strings"
)

type CacheCommand struct {
	proto       *api.Command
	instruction *api.CacheInstruction

	reuploadOnChangesExplicitlySet bool

	parseable.DefaultParser
}

func NewCacheCommand(mergedEnv map[string]string, boolevator *boolevator.Boolevator) *CacheCommand {
	cache := &CacheCommand{
		proto: &api.Command{},
		instruction: &api.CacheInstruction{
			ReuploadOnChanges: true,
		},
	}

	cache.RequiredField(nameable.NewSimpleNameable("folder"), schema.TodoSchema, func(node *node.Node) error {
		folder, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		// https://github.com/cirruslabs/cirrus-ci-agent/issues/47
		const homePrefix = "~/"
		folder = strings.TrimSpace(folder)
		if strings.HasPrefix(folder, homePrefix) {
			folder = "$HOME/" + strings.TrimPrefix(folder, homePrefix)
		}

		cache.instruction.Folder = folder

		return nil
	})

	cache.OptionalField(nameable.NewSimpleNameable("fingerprint_script"), schema.TodoSchema, func(node *node.Node) error {
		scripts, err := node.GetScript()
		if err != nil {
			return err
		}

		cache.instruction.FingerprintScripts = scripts

		// Disable the default "dumb" re-upload behavior unless otherwise specified by the user since
		// we now have a better way of figuring out whether we need to upload the cache or not
		if !cache.reuploadOnChangesExplicitlySet {
			cache.instruction.ReuploadOnChanges = false
		}

		return nil
	})

	cache.OptionalField(nameable.NewSimpleNameable("populate_script"), schema.TodoSchema, func(node *node.Node) error {
		scripts, err := node.GetScript()
		if err != nil {
			return err
		}
		cache.instruction.PopulateScripts = scripts
		return nil
	})

	cache.OptionalField(nameable.NewSimpleNameable("reupload_on_changes"), schema.TodoSchema, func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(mergedEnv, boolevator)
		if err != nil {
			return err
		}

		cache.instruction.ReuploadOnChanges = evaluation
		cache.reuploadOnChangesExplicitlySet = true

		return nil
	})

	return cache
}

func (cache *CacheCommand) Parse(node *node.Node) error {
	if err := cache.DefaultParser.Parse(node); err != nil {
		return err
	}

	cacheNameable := nameable.NewRegexNameable("^(.*)cache$")
	cache.proto.Name = cacheNameable.FirstGroupOrDefault(node.Name, "main")

	return nil
}

func (cache *CacheCommand) Proto() *api.Command {
	cache.proto.Instruction = &api.Command_CacheInstruction{
		CacheInstruction: cache.instruction,
	}

	return cache.proto
}
