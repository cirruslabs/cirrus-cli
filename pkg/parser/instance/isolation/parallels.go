package isolation

import (
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parserkit"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	jsschema "github.com/lestrrat-go/jsschema"
	"strings"
)

type Parallels struct {
	proto *api.Isolation_Parallels_

	parseable.DefaultParser
}

func NewParallels(mergedEnv map[string]string) *Parallels {
	parallels := &Parallels{
		proto: &api.Isolation_Parallels_{
			Parallels: &api.Isolation_Parallels{},
		},
	}

	imageSchema := schema.String("Image name.")
	parallels.OptionalField(nameable.NewSimpleNameable("image"), imageSchema, func(node *node.Node) error {
		image, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		parallels.proto.Parallels.Image = image

		return nil
	})

	userSchema := schema.String("SSH username")
	parallels.OptionalField(nameable.NewSimpleNameable("user"), userSchema, func(node *node.Node) error {
		user, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		parallels.proto.Parallels.User = user

		return nil
	})

	passwordSchema := schema.String("SSH password")
	parallels.OptionalField(nameable.NewSimpleNameable("password"), passwordSchema, func(node *node.Node) error {
		password, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		parallels.proto.Parallels.Password = password

		return nil
	})

	platformSchema := schema.Platform("Image Platform.")
	parallels.OptionalField(nameable.NewSimpleNameable("platform"), platformSchema, func(node *node.Node) error {
		platform, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		resolvedPlatform, ok := api.Platform_value[strings.ToUpper(platform)]
		if !ok {
			return node.ParserError("unsupported platform name: %q", platform)
		}

		parallels.proto.Parallels.Platform = api.Platform(resolvedPlatform)

		return nil
	})

	return parallels
}

func (parallels *Parallels) Parse(node *node.Node, parserKit *parserkit.ParserKit) error {
	return parallels.DefaultParser.Parse(node, parserKit)
}

func (parallels *Parallels) Proto() *api.Isolation_Parallels_ {
	return parallels.proto
}

func (parallels *Parallels) Schema() *jsschema.Schema {
	modifiedSchema := parallels.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "Parallels VM isolation."

	return modifiedSchema
}
