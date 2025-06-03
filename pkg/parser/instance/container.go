package instance

import (
	"strconv"

	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance/resources"
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
	proto *api.ContainerInstance

	parseable.DefaultParser
}

func NewCommunityContainer(
	mergedEnv map[string]string,
	architecture api.Architecture,
	parserKit *parserkit.ParserKit,
) *Container {
	container := &Container{
		proto: &api.ContainerInstance{
			Architecture: architecture,
		},
	}

	imageSchema := schema.String("Docker Image to use.")
	container.OptionalField(nameable.NewSimpleNameable("image"), imageSchema, func(node *node.Node) error {
		// reset dockerfile as CI environment
		container.proto.Dockerfile = ""
		container.proto.DockerArguments = nil

		image, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		container.proto.Image = image
		return nil
	})

	dockerfileSchema := schema.String("Relative path to Dockerfile to build container from.")
	container.OptionalField(nameable.NewSimpleNameable("dockerfile"), dockerfileSchema, func(node *node.Node) error {
		dockerfile, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		container.proto.Dockerfile = dockerfile
		return nil
	})

	dockerArgumentsNameable := nameable.NewSimpleNameable("docker_arguments")
	dockerArgumentsSchema := schema.Map("Arguments for Docker build")
	container.OptionalField(dockerArgumentsNameable, dockerArgumentsSchema, func(node *node.Node) error {
		dockerArguments, err := node.GetMapOrListOfMapsWithExpansion(mergedEnv)
		if err != nil {
			return err
		}
		container.proto.DockerArguments = dockerArguments
		return nil
	})

	container.OptionalField(nameable.NewSimpleNameable("cpu"), schema.Number(""), func(node *node.Node) error {
		cpu, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		cpuFloat, err := strconv.ParseFloat(cpu, 32)
		if err != nil {
			return err
		}
		container.proto.Cpu = float32(cpuFloat)
		return nil
	})

	container.OptionalField(nameable.NewSimpleNameable("memory"), schema.Memory(), func(node *node.Node) error {
		memory, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		memoryParsed, err := resources.ParseMegaBytes(memory)
		if err != nil {
			return node.ParserError("%s", err.Error())
		}
		container.proto.Memory = uint32(memoryParsed)
		return nil
	})

	additionalContainersNameable := nameable.NewSimpleNameable("additional_containers")
	acSchema := schema.ArrayOf(NewAdditionalContainer(nil, nil).Schema())
	container.OptionalField(additionalContainersNameable, acSchema, func(node *node.Node) error {
		for _, child := range node.Children {
			ac := NewAdditionalContainer(mergedEnv, parserKit)
			additionalContainer, err := ac.Parse(child, parserKit)
			if err != nil {
				return err
			}
			container.proto.AdditionalContainers = append(container.proto.AdditionalContainers, additionalContainer)
		}
		return nil
	})

	// no-op
	container.OptionalField(nameable.NewSimpleNameable("kvm"), schema.Condition(""), func(node *node.Node) error {
		return nil
	})

	// no-op
	container.OptionalField(nameable.NewSimpleNameable("registry_config"), schema.String(""), func(node *node.Node) error {
		return nil
	})

	// no-op
	inMemorySchema := schema.Condition("")
	container.OptionalField(nameable.NewSimpleNameable("use_in_memory_disk"), inMemorySchema, func(node *node.Node) error {
		return nil
	})

	// no-op
	sipSchema := schema.Condition("")
	container.OptionalField(nameable.NewSimpleNameable("use_static_ip"), sipSchema, func(node *node.Node) error {
		return nil
	})

	return container
}

func (container *Container) Parse(node *node.Node, parserKit *parserkit.ParserKit) (*api.ContainerInstance, error) {
	if err := container.DefaultParser.Parse(node, parserKit); err != nil {
		return nil, err
	}

	// Resource defaults
	if container.proto.Cpu == 0 {
		container.proto.Cpu = defaultCPU
	}
	if container.proto.Memory == 0 {
		container.proto.Memory = defaultMemory
	}

	return container.proto, nil
}

func (container *Container) Schema() *jsschema.Schema {
	modifiedSchema := container.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "Container definition for Community Cluster."

	return modifiedSchema
}
