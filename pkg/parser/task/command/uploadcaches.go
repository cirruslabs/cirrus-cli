package command

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	jsschema "github.com/lestrrat-go/jsschema"
)

func UploadCachesHelper(
	env map[string]string,
	previousCommands []*api.Command,
	node *node.Node,
) ([]*api.Command, error) {
	var result []*api.Command

	cacheNames, err := node.GetSliceOfExpandedStrings(env)
	if err != nil {
		return nil, err
	}

	for _, cacheName := range cacheNames {
		var isCacheNameDefined bool

		for _, command := range previousCommands {
			if command.Name == cacheName {
				isCacheNameDefined = true
				break
			}
		}

		if !isCacheNameDefined {
			return nil, node.ParserError("no cache with name %q is defined", cacheName)
		}

		var hasCacheUploadCommand bool
		cacheUploadCommandName := fmt.Sprintf("Upload '%s' cache", cacheName)

		for _, command := range previousCommands {
			if command.Name == cacheUploadCommandName {
				hasCacheUploadCommand = true
				break
			}
		}

		if hasCacheUploadCommand {
			return nil, node.ParserError("the cache with name %q is already scheduled to be uploaded in one "+
				"of the previous upload_caches instructions", cacheUploadCommandName)
		}

		result = append(result, &api.Command{
			Name: cacheUploadCommandName,
			Instruction: &api.Command_UploadCacheInstruction{
				UploadCacheInstruction: &api.UploadCacheInstruction{
					CacheName: cacheName,
				},
			},
		})
	}

	return result, nil
}

func UploadCachesSchema() *jsschema.Schema {
	return schema.ArrayOf(schema.String("Cache name to upload."))
}
