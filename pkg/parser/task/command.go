package task

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
)

type CacheCommand struct {
	proto       *api.Command
	instruction *api.CacheInstruction

	parseable.DefaultParser
}

func NewCacheCommand() *CacheCommand {
	cache := &CacheCommand{
		proto:       &api.Command{},
		instruction: &api.CacheInstruction{},
	}

	cache.RequiredField(nameable.NewSimpleNameable("folder"), schema.TodoSchema, func(node *node.Node) error {
		folder, err := node.GetExpandedStringValue(nil)
		if err != nil {
			return err
		}
		cache.instruction.Folder = folder
		return nil
	})

	cache.OptionalField(nameable.NewSimpleNameable("fingerprint_script"), schema.TodoSchema, func(node *node.Node) error {
		scripts, err := node.GetSliceOfNonEmptyStrings()
		if err != nil {
			return err
		}
		cache.instruction.FingerprintScripts = scripts
		return nil
	})

	cache.OptionalField(nameable.NewSimpleNameable("populate_script"), schema.TodoSchema, func(node *node.Node) error {
		scripts, err := node.GetSliceOfNonEmptyStrings()
		if err != nil {
			return err
		}
		cache.instruction.PopulateScripts = scripts
		return nil
	})

	return cache
}

func (cache *CacheCommand) Parse(node *node.Node) error {
	if err := cache.DefaultParser.Parse(node); err != nil {
		return err
	}

	cacheNameable := nameable.NewRegexNameable("(.*)cache")
	cache.proto.Name = cacheNameable.FirstGroupOrDefault(node.Name, "main")

	return nil
}

func (cache *CacheCommand) Proto() *api.Command {
	cache.proto.Instruction = &api.Command_CacheInstruction{
		CacheInstruction: cache.instruction,
	}

	return cache.proto
}
