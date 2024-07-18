package vetu

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

type Vetu struct {
	proto *api.Isolation_Vetu_

	parseable.DefaultParser
}

func New(mergedEnv map[string]string, parserKit *parserkit.ParserKit) *Vetu {
	vetu := &Vetu{
		proto: &api.Isolation_Vetu_{
			Vetu: &api.Isolation_Vetu{
				User:     "admin",
				Password: "admin",
			},
		},
	}

	vmSchema := schema.String("Source VM image (or name) to clone the new VM from.")
	vetu.OptionalField(nameable.NewSimpleNameable("image"), vmSchema, func(node *node.Node) error {
		image, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		vetu.proto.Vetu.Image = image

		return nil
	})

	userSchema := schema.StringWithDefaultValue("SSH username.", vetu.proto.Vetu.User)
	vetu.OptionalField(nameable.NewSimpleNameable("user"), userSchema, func(node *node.Node) error {
		user, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		vetu.proto.Vetu.User = user

		return nil
	})

	passwordSchema := schema.StringWithDefaultValue("SSH password.", vetu.proto.Vetu.Password)
	vetu.OptionalField(nameable.NewSimpleNameable("password"), passwordSchema, func(node *node.Node) error {
		password, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		vetu.proto.Vetu.Password = password

		return nil
	})

	portSchema := schema.Integer("SSH port.")
	vetu.OptionalField(nameable.NewSimpleNameable("port"), portSchema, func(node *node.Node) error {
		rawPort, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		port, err := strconv.ParseUint(rawPort, 10, 16)
		if err != nil {
			return node.ParserError("failed to parse \"port:\" value: %v", err)
		}

		vetu.proto.Vetu.Port = uint32(port)

		return nil
	})

	cpuSchema := schema.Number("Number of VM CPUs.")
	vetu.OptionalField(nameable.NewSimpleNameable("cpu"), cpuSchema, func(node *node.Node) error {
		cpu, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		cpuParsed, err := strconv.ParseUint(cpu, 10, 32)
		if err != nil {
			return node.ParserError("%s", err.Error())
		}
		vetu.proto.Vetu.Cpu = uint32(cpuParsed)
		return nil
	})

	memorySchema := schema.Memory()
	memorySchema.Description = "VM memory size in megabytes."
	vetu.OptionalField(nameable.NewSimpleNameable("memory"), memorySchema, func(node *node.Node) error {
		memory, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		memoryParsed, err := resources.ParseMegaBytes(memory)
		if err != nil {
			return node.ParserError("%s", err.Error())
		}
		vetu.proto.Vetu.Memory = uint32(memoryParsed)
		return nil
	})

	vetu.OptionalField(nameable.NewSimpleNameable("networking"), nil, func(node *node.Node) error {
		if node.HasChild("host") {
			vetu.proto.Vetu.Networking = &api.Isolation_Vetu_Host_{
				Host: &api.Isolation_Vetu_Host{},
			}
		} else {
			return node.ParserError("please specify the networking to use")
		}

		return nil
	})

	diskSizeSchema := schema.Integer("Disk size to use in gigabytes.")
	vetu.OptionalField(nameable.NewSimpleNameable("disk_size"), diskSizeSchema, func(node *node.Node) error {
		diskSizeRaw, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		diskSize, err := strconv.ParseUint(diskSizeRaw, 10, 16)
		if err != nil {
			return node.ParserError("%v", err)
		}

		vetu.proto.Vetu.DiskSize = uint32(diskSize)

		return nil
	})

	return vetu
}

func (vetu *Vetu) Parse(node *node.Node, parserKit *parserkit.ParserKit) error {
	return vetu.DefaultParser.Parse(node, parserKit)
}

func (vetu *Vetu) Proto() *api.Isolation_Vetu_ {
	return vetu.proto
}

func (vetu *Vetu) Schema() *jsschema.Schema {
	modifiedSchema := vetu.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "Vetu VM isolation."

	return modifiedSchema
}
