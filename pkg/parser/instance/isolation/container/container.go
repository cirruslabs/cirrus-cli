package container

import (
	"strconv"
	"strings"

	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/constants"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance/resources"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance/volume"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parserkit"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	jsschema "github.com/lestrrat-go/jsschema"
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
	container.OptionalField(nameable.NewSimpleNameable("image"), imageSchema, func(node *node.Node) error {
		// reset dockerfile as CI environment
		container.proto.Container.Dockerfile = ""

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

		for _, v := range volumes {
			v, err := volume.ParseVolume(node, v)
			if err != nil {
				return err
			}
			container.proto.Container.Volumes = append(container.proto.Container.Volumes, v)
		}

		return nil
	})

	dockerfileSchema := schema.String("Relative path to Dockerfile to build container from.")
	container.OptionalField(nameable.NewSimpleNameable("dockerfile"), dockerfileSchema, func(node *node.Node) error {
		dockerfile, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		// Guard against container image collision risk that arises when using Dockerfile
		// with no architecture. For more details see issue[1] and comment[2].
		//
		// [1]: https://github.com/cirruslabs/cirrus-cli/issues/550
		// [2]: https://github.com/cirruslabs/cirrus-cli/pull/545#issuecomment-1224597905
		if mergedEnv[constants.EnvironmentCirrusArch] == "" {
			return node.ParserError("container with \"dockerfile:\" also needs" +
				" a CIRRUS_ARCH environment variable to be specified")
		}

		container.proto.Container.Dockerfile = dockerfile

		return nil
	})

	dockerArgumentsNameable := nameable.NewSimpleNameable("docker_arguments")
	dockerArgumentsSchema := schema.Map("Arguments for Docker build.")
	container.OptionalField(dockerArgumentsNameable, dockerArgumentsSchema, func(node *node.Node) error {
		dockerArguments, err := node.GetMapOrListOfMapsWithExpansion(mergedEnv)
		if err != nil {
			return err
		}
		container.proto.Container.DockerArguments = dockerArguments
		return nil
	})

	platformSchema := schema.Platform("Image Platform.")
	container.OptionalField(nameable.NewSimpleNameable("platform"), platformSchema, func(node *node.Node) error {
		platform, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		resolvedPlatform, ok := api.Platform_value[strings.ToUpper(platform)]
		if !ok {
			return node.ParserError("unsupported platform name: %q", platform)
		}

		container.proto.Container.Platform = api.Platform(resolvedPlatform)

		return nil
	})

	return container
}

func (container *Container) Parse(node *node.Node, parserKit *parserkit.ParserKit) error {
	if err := container.DefaultParser.Parse(node, parserKit); err != nil {
		return err
	}

	// Resource defaults
	if container.proto.Container.Cpu == 0 {
		container.proto.Container.Cpu = defaultCPU
	}
	if container.proto.Container.Memory == 0 {
		container.proto.Container.Memory = defaultMemory
	}

	// Finally, remove the Docker arguments if "dockerfile:"
	// was not specified or was overridden by "image:"
	if container.proto.Container.Dockerfile == "" {
		container.proto.Container.DockerArguments = nil
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
