package instance

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance/resources"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parserkit"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	jsschema "github.com/lestrrat-go/jsschema"
	"strconv"
)

type MacOSInstance struct {
	proto *api.MacOSInstance

	parseable.DefaultParser
}

func NewMacOSInstance(mergedEnv map[string]string, parserKit *parserkit.ParserKit) *MacOSInstance {
	instance := &MacOSInstance{
		proto: &api.MacOSInstance{
			User:     "admin",
			Password: "admin",
			Cpu:      4,
			Memory:   8192,
		},
	}

	imageSchema := schema.String("Tart Image to use.")
	instance.OptionalField(nameable.NewSimpleNameable("image"), imageSchema, func(node *node.Node) error {
		image, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		instance.proto.Image = image
		return nil
	})

	userSchema := schema.String("username for SSH connection.")
	instance.OptionalField(nameable.NewSimpleNameable("user"), userSchema, func(node *node.Node) error {
		user, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		instance.proto.User = user
		return nil
	})

	passwordSchema := schema.String("password for SSH connection.")
	instance.OptionalField(nameable.NewSimpleNameable("password"), passwordSchema, func(node *node.Node) error {
		password, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		instance.proto.Password = password
		return nil
	})

	instance.OptionalField(nameable.NewSimpleNameable("cpu"), schema.Number(""), func(node *node.Node) error {
		cpu, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		cpuParsed, err := strconv.ParseUint(cpu, 10, 16)
		if err != nil {
			return err
		}
		instance.proto.Cpu = uint32(cpuParsed)
		return nil
	})

	instance.OptionalField(nameable.NewSimpleNameable("memory"), schema.Memory(), func(node *node.Node) error {
		memory, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		memoryParsed, err := resources.ParseMegaBytes(memory)
		if err != nil {
			return node.ParserError("%s", err.Error())
		}
		instance.proto.Memory = uint32(memoryParsed)
		return nil
	})

	return instance
}

func (instance *MacOSInstance) Parse(node *node.Node, parserKit *parserkit.ParserKit) (*api.MacOSInstance, error) {
	if err := instance.DefaultParser.Parse(node, parserKit); err != nil {
		return nil, err
	}

	return instance.proto, nil
}

func (instance *MacOSInstance) Schema() *jsschema.Schema {
	modifiedSchema := instance.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "MacOS VM definition."

	return modifiedSchema
}
