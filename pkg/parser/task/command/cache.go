package command

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parserkit"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	jsschema "github.com/lestrrat-go/jsschema"
	"strings"
)

type CacheCommand struct {
	proto       *api.Command
	instruction *api.CacheInstruction

	reuploadOnChangesExplicitlySet bool
	folder                         string
	folders                        []string

	parseable.DefaultParser
}

func NewCacheCommand(mergedEnv map[string]string, parserKit *parserkit.ParserKit) *CacheCommand {
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
	cache.OptionalField(nameable.NewSimpleNameable("folder"), folderSchema, func(node *node.Node) error {
		folder, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		cache.folder = folder

		return nil
	})

	foldersSchema := schema.StringOrListOfStrings("A list of folders to cache.")
	cache.OptionalField(nameable.NewSimpleNameable("folders"), foldersSchema, func(node *node.Node) error {
		folders, err := node.GetSliceOfExpandedStrings(mergedEnv)
		if err != nil {
			return err
		}

		cache.folders = folders

		return nil
	})

	fpScriptSchema := schema.Script("Script that is used to calculate cache key.")
	cache.OptionalField(nameable.NewSimpleNameable("fingerprint_script"), fpScriptSchema, func(node *node.Node) error {
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

	fpKeySchema := schema.String("Cache key in it's raw form.")
	cache.OptionalField(nameable.NewSimpleNameable("fingerprint_key"), fpKeySchema, func(node *node.Node) error {
		key, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		cache.instruction.FingerprintKey = key

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
		evaluation, err := node.GetBoolValue(mergedEnv, parserKit.Boolevator)
		if err != nil {
			return err
		}

		cache.instruction.ReuploadOnChanges = evaluation
		cache.reuploadOnChangesExplicitlySet = true

		return nil
	})

	optimisticRestoreSchema := schema.Condition("A flag to enable optimistic cache restoration on miss by finding the latest available key.")
	cache.OptionalField(nameable.NewSimpleNameable("optimistically_restore_on_miss"), optimisticRestoreSchema, func(node *node.Node) error {
		evaluation, err := node.GetBoolValue(mergedEnv, parserKit.Boolevator)
		if err != nil {
			return err
		}

		cache.instruction.OptimisticallyRestoreOnMiss = evaluation

		return nil
	})

	return cache
}

func (cache *CacheCommand) Parse(node *node.Node, parserKit *parserkit.ParserKit) error {
	if err := cache.DefaultParser.Parse(node, parserKit); err != nil {
		return err
	}

	if cache.proto.Name == "" {
		cacheNameable := nameable.NewRegexNameable("^(.*)cache$")
		cache.proto.Name = cacheNameable.FirstGroupOrDefault(node.Name, "main")
	}

	// Deal with cache folders
	if cache.folder == "" && len(cache.folders) == 0 {
		return node.ParserError("please specify the folders to cache, with either folder: or folders:")
	} else if cache.folder != "" && len(cache.folders) != 0 {
		return node.ParserError("please specify either folder: or folders: but not both, to avoid ambiguity")
	}

	var folders []string

	if cache.folder != "" {
		folders = []string{cache.folder}
	} else {
		folders = cache.folders
	}

	cache.instruction.Folders = fixFolders(folders)

	// fingerprint_script and fingerprint_key are mutually exclusive,
	// ensure that we don't allow such ambiguities as early as possible
	// (in terms of build pipeline, not this method)
	if len(cache.instruction.FingerprintScripts) != 0 && cache.instruction.FingerprintKey != "" {
		return node.ParserError("please either use fingerprint_script: or fingerprint_key, since otherwise " +
			"there's ambiguity about which one to prefer for cache key calculation")
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

	modifiedSchema.OneOf = jsschema.SchemaList{
		&jsschema.Schema{
			Required:             []string{"folder"},
			AdditionalItems:      &jsschema.AdditionalItems{Schema: nil},
			AdditionalProperties: &jsschema.AdditionalProperties{Schema: nil},
		},
		&jsschema.Schema{
			Required:             []string{"folders"},
			AdditionalItems:      &jsschema.AdditionalItems{Schema: nil},
			AdditionalProperties: &jsschema.AdditionalProperties{Schema: nil},
		},
	}

	return modifiedSchema
}

func GenUploadCacheCmds(commands []*api.Command) (result []*api.Command) {
	// Don't generate any cache upload instructions if "upload_caches" instruction was used,
	// in which case it'd insert at least one cache upload instruction.
	for _, command := range commands {
		if _, ok := command.Instruction.(*api.Command_UploadCacheInstruction); ok {
			return []*api.Command{}
		}
	}

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

func fixFolders(folders []string) (result []string) {
	for _, folder := range folders {
		// https://github.com/cirruslabs/cirrus-ci-agent/issues/47
		const homePrefix = "~/"
		folder = strings.TrimSpace(folder)
		if strings.HasPrefix(folder, homePrefix) {
			folder = "$HOME/" + strings.TrimPrefix(folder, homePrefix)
		}

		result = append(result, folder)
	}

	return
}
