package instance

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance/resources"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	jsschema "github.com/lestrrat-go/jsschema"
	"strconv"
)

type WindowsContainer struct {
	proto *api.ContainerInstance

	parseable.DefaultParser
}

func NewWindowsCommunityContainer(mergedEnv map[string]string, boolevator *boolevator.Boolevator) *WindowsContainer {
	container := &WindowsContainer{
		proto: &api.ContainerInstance{
			Platform:  api.Platform_WINDOWS,
			OsVersion: "2019",
		},
	}

	imageSchema := schema.String("Docker Image to use.")
	container.OptionalField(nameable.NewSimpleNameable("image"), imageSchema, func(node *node.Node) error {
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
		dockerArguments, err := node.GetStringMapping()
		if err != nil {
			return err
		}
		container.proto.DockerArguments = dockerArguments
		return nil
	})

	osVersionSchema := schema.Enum([]interface{}{"2019", "1709", "1803"}, "Windows version of container.")
	container.OptionalField(nameable.NewSimpleNameable("os_version"), osVersionSchema, func(node *node.Node) error {
		osVersion, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		container.proto.OsVersion = osVersion

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

	// no-op
	sipSchema := schema.Condition("")
	container.OptionalField(nameable.NewSimpleNameable("use_static_ip"), sipSchema, func(node *node.Node) error {
		return nil
	})

	return container
}

func (container *WindowsContainer) Parse(node *node.Node) (*api.ContainerInstance, error) {
	if err := container.DefaultParser.Parse(node); err != nil {
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

func (container *WindowsContainer) Schema() *jsschema.Schema {
	modifiedSchema := container.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "Windows Container definition for Community Cluster."

	return modifiedSchema
}
