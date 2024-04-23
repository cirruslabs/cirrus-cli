package tart

import (
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance/resources"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parserkit"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	jsschema "github.com/lestrrat-go/jsschema"
	"strconv"
)

type Tart struct {
	proto *api.Isolation_Tart_

	parseable.DefaultParser
}

func New(mergedEnv map[string]string, parserKit *parserkit.ParserKit) *Tart {
	tart := &Tart{
		proto: &api.Isolation_Tart_{
			Tart: &api.Isolation_Tart{},
		},
	}

	vmSchema := schema.String("Source VM image (or name) to clone the new VM from.")
	tart.OptionalField(nameable.NewSimpleNameable("image"), vmSchema, func(node *node.Node) error {
		image, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		tart.proto.Tart.Image = image

		return nil
	})

	userSchema := schema.String("SSH username.")
	tart.OptionalField(nameable.NewSimpleNameable("user"), userSchema, func(node *node.Node) error {
		user, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		tart.proto.Tart.User = user

		return nil
	})

	passwordSchema := schema.String("SSH password.")
	tart.OptionalField(nameable.NewSimpleNameable("password"), passwordSchema, func(node *node.Node) error {
		password, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		tart.proto.Tart.Password = password

		return nil
	})

	portSchema := schema.Integer("SSH port.")
	tart.OptionalField(nameable.NewSimpleNameable("port"), portSchema, func(node *node.Node) error {
		rawPort, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		port, err := strconv.ParseUint(rawPort, 10, 16)
		if err != nil {
			return node.ParserError("failed to parse \"port:\" value: %v", err)
		}

		tart.proto.Tart.Port = uint32(port)

		return nil
	})

	cpuSchema := schema.Number("Number of VM CPUs.")
	tart.OptionalField(nameable.NewSimpleNameable("cpu"), cpuSchema, func(node *node.Node) error {
		cpu, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		cpuParsed, err := strconv.ParseUint(cpu, 10, 32)
		if err != nil {
			return node.ParserError("%s", err.Error())
		}
		tart.proto.Tart.Cpu = uint32(cpuParsed)
		return nil
	})

	memorySchema := schema.Memory()
	memorySchema.Description = "VM memory size in megabytes."
	tart.OptionalField(nameable.NewSimpleNameable("memory"), memorySchema, func(node *node.Node) error {
		memory, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		memoryParsed, err := resources.ParseMegaBytes(memory)
		if err != nil {
			return node.ParserError("%s", err.Error())
		}
		tart.proto.Tart.Memory = uint32(memoryParsed)
		return nil
	})

	softnetSchema := schema.Boolean("Enable or disable the Softnet networking.")
	tart.OptionalField(nameable.NewSimpleNameable("softnet"), softnetSchema, func(node *node.Node) error {
		softnet, err := node.GetBoolValue(mergedEnv, parserKit.Boolevator)
		if err != nil {
			return err
		}
		tart.proto.Tart.Softnet = softnet
		return nil
	})

	displaySchema := schema.String("Virtual display configuration.")
	tart.OptionalField(nameable.NewSimpleNameable("display"), displaySchema, func(node *node.Node) error {
		display, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		tart.proto.Tart.Display = display

		return nil
	})

	volumeSchema := schema.ArrayOf(NewVolume(mergedEnv, parserKit).Schema())
	tart.OptionalField(nameable.NewSimpleNameable("volumes"), volumeSchema, func(node *node.Node) error {
		for _, child := range node.Children {
			volume, err := NewVolume(mergedEnv, parserKit).Parse(child, parserKit)
			if err != nil {
				return err
			}

			tart.proto.Tart.Volumes = append(tart.proto.Tart.Volumes, volume)
		}

		return nil
	})

	return tart
}

func (tart *Tart) Parse(node *node.Node, parserKit *parserkit.ParserKit) error {
	return tart.DefaultParser.Parse(node, parserKit)
}

func (tart *Tart) Proto() *api.Isolation_Tart_ {
	return tart.proto
}

func (tart *Tart) Schema() *jsschema.Schema {
	modifiedSchema := tart.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "Tart VM isolation."

	return modifiedSchema
}
