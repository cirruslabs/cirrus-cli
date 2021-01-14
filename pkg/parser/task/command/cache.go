package command

import (
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	jsschema "github.com/lestrrat-go/jsschema"
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

	cache.OptionalField(nameable.NewSimpleNameable("name"), schema.String(""), func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		cache.proto.Name = name
		return nil
	})

	folderSchema := schema.String("Path of a folder to cache.")
	cache.RequiredField(nameable.NewSimpleNameable("folder"), folderSchema, func(node *node.Node) error {
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

	fpSchema := schema.Script("Script that is used to calculate cache key.")
	cache.OptionalField(nameable.NewSimpleNameable("fingerprint_script"), fpSchema, func(node *node.Node) error {
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

	populateSchema := schema.Script("In case of a cache miss this script will be executed.")
	cache.OptionalField(nameable.NewSimpleNameable("populate_script"), populateSchema, func(node *node.Node) error {
		scripts, err := node.GetScript()
		if err != nil {
			return err
		}
		cache.instruction.PopulateScripts = scripts
		return nil
	})

	reuploadSchema := schema.Condition("A flag to check if contents of folder has changed after a cache hit.")
	cache.OptionalField(nameable.NewSimpleNameable("reupload_on_changes"), reuploadSchema, func(node *node.Node) error {
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

	if cache.proto.Name == "" {
		cacheNameable := nameable.NewRegexNameable("^(.*)cache$")
		cache.proto.Name = cacheNameable.FirstGroupOrDefault(node.Name, "main")
	}

	return nil
}

func (cache *CacheCommand) Proto() *api.Command {
	cache.proto.Instruction = &api.Command_CacheInstruction{
		CacheInstruction: cache.instruction,
	}

	return cache.proto
}

func (cache *CacheCommand) Schema() *jsschema.Schema {
	modifiedSchema := cache.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "Folder Cache Definition."

	return modifiedSchema
}

func GenUploadCacheCmds(commands []*api.Command) (result []*api.Command) {
	for _, command := range commands {
		_, ok := command.Instruction.(*api.Command_CacheInstruction)
		if !ok {
			continue
		}

		uploadCommand := &api.Command{
			Name: fmt.Sprintf("Upload '%s' cache", command.Name),
			Instruction: &api.Command_UploadCacheInstruction{
				UploadCacheInstruction: &api.UploadCacheInstruction{
					CacheName: command.Name,
				},
			},
			ExecutionBehaviour: command.ExecutionBehaviour,
		}

		result = append(result, uploadCommand)
	}

	return
}
