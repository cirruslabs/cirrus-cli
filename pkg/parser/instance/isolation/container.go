package isolation

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance/resources"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	jsschema "github.com/lestrrat-go/jsschema"
	"strconv"
	"strings"
)

const (
	sourceSplitIdx = iota
	targetSplitIdx
	flagsSplitIdx
)

const (
	numSourceAndTargetSplits      = 2
	numSourceTargetAndFlagsSplits = 3
)

const (
	defaultCPU    = 2.0
	defaultMemory = 4096
)

type Container struct {
	proto *api.Isolation_Container_

	parseable.DefaultParser
}

func NewContainer(mergedEnv map[string]string) *Container {
	container := &Container{
		proto: &api.Isolation_Container_{
			Container: &api.Isolation_Container{},
		},
	}

	imageSchema := schema.String("Container image to use.")
	container.RequiredField(nameable.NewSimpleNameable("image"), imageSchema, func(node *node.Node) error {
		image, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		container.proto.Container.Image = image

		return nil
	})

	cpuSchema := schema.Number("CPU units for the container to use.")
	container.OptionalField(nameable.NewSimpleNameable("cpu"), cpuSchema, func(node *node.Node) error {
		cpu, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		cpuFloat, err := strconv.ParseFloat(cpu, 32)
		if err != nil {
			return err
		}
		container.proto.Container.Cpu = float32(cpuFloat)
		return nil
	})

	memorySchema := schema.Memory()
	memorySchema.Description = "Memory in megabytes for the container to use."
	container.OptionalField(nameable.NewSimpleNameable("memory"), memorySchema, func(node *node.Node) error {
		memory, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		memoryParsed, err := resources.ParseMegaBytes(memory)
		if err != nil {
			return node.ParserError("%s", err.Error())
		}
		container.proto.Container.Memory = uint32(memoryParsed)
		return nil
	})

	container.OptionalField(nameable.NewSimpleNameable("volumes"), schema.Volumes(), func(node *node.Node) error {
		volumes, err := node.GetSliceOfExpandedStrings(mergedEnv)
		if err != nil {
			return err
		}

		for _, volume := range volumes {
			splits := strings.Split(volume, ":")

			switch len(splits) {
			case numSourceAndTargetSplits:
				// src:dst
				container.proto.Container.Volumes = append(container.proto.Container.Volumes, &api.Volume{
					Source: splits[sourceSplitIdx],
					Target: splits[targetSplitIdx],
				})
			case numSourceTargetAndFlagsSplits:
				// src:dst:ro
				if splits[flagsSplitIdx] != "ro" {
					return node.ParserError("only \"ro\" volume flag is currently supported")
				}
				container.proto.Container.Volumes = append(container.proto.Container.Volumes, &api.Volume{
					Source:   splits[sourceSplitIdx],
					Target:   splits[targetSplitIdx],
					ReadOnly: true,
				})
			default:
				return node.ParserError("only source:target[:ro] volume specification is currently supported")
			}
		}

		return nil
	})

	return container
}

func (container *Container) Parse(node *node.Node) error {
	if err := container.DefaultParser.Parse(node); err != nil {
		return err
	}

	// Resource defaults
	if container.proto.Container.Cpu == 0 {
		container.proto.Container.Cpu = defaultCPU
	}
	if container.proto.Container.Memory == 0 {
		container.proto.Container.Memory = defaultMemory
	}

	return nil
}

func (container *Container) Proto() *api.Isolation_Container_ {
	return container.proto
}

func (container *Container) Schema() *jsschema.Schema {
	modifiedSchema := container.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "Container engine isolation."

	return modifiedSchema
}
