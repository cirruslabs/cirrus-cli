package instance

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"strconv"
)

const (
	defaultCPU    = 2.0
	defaultMemory = 4096
)

type Container struct {
	proto *api.ContainerInstance

	parseable.DefaultParser
}

func NewCommunityContainer(mergedEnv map[string]string, boolevator *boolevator.Boolevator) *Container {
	container := &Container{
		proto: &api.ContainerInstance{},
	}

	container.OptionalField(nameable.NewSimpleNameable("image"), schema.TodoSchema, func(node *node.Node) error {
		image, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		container.proto.Image = image
		return nil
	})

	container.OptionalField(nameable.NewSimpleNameable("dockerfile"), schema.TodoSchema, func(node *node.Node) error {
		dockerfile, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		container.proto.DockerfilePath = dockerfile
		return nil
	})

	dockerArgumentsNameable := nameable.NewSimpleNameable("docker_arguments")
	container.OptionalField(dockerArgumentsNameable, schema.TodoSchema, func(node *node.Node) error {
		dockerArguments, err := node.GetStringMapping()
		if err != nil {
			return err
		}
		container.proto.DockerArguments = dockerArguments
		return nil
	})

	container.OptionalField(nameable.NewSimpleNameable("cpu"), schema.TodoSchema, func(node *node.Node) error {
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

	container.OptionalField(nameable.NewSimpleNameable("memory"), schema.TodoSchema, func(node *node.Node) error {
		memory, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		memoryParsed, err := ParseMegaBytes(memory)
		if err != nil {
			return err
		}
		container.proto.Memory = uint32(memoryParsed)
		return nil
	})

	additionalContainersNameable := nameable.NewSimpleNameable("additional_containers")
	container.OptionalField(additionalContainersNameable, schema.TodoSchema, func(node *node.Node) error {
		for _, child := range node.Children {
			ac := NewAdditionalContainer(mergedEnv, boolevator)
			additionalContainer, err := ac.Parse(child)
			if err != nil {
				return err
			}
			container.proto.AdditionalContainers = append(container.proto.AdditionalContainers, additionalContainer)
		}
		return nil
	})

	return container
}

func (container *Container) Parse(node *node.Node) (*api.ContainerInstance, error) {
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
