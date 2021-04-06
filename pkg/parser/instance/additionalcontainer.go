package instance

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	jsschema "github.com/lestrrat-go/jsschema"
	"strconv"
	"strings"
	"unicode"
)

const (
	defaultAdditionalCPU    = 0.5
	defaultAdditionalMemory = 512
)

type AdditionalContainer struct {
	proto *api.AdditionalContainer

	parseable.DefaultParser
}

func parsePort(port string) (*api.PortMapping, error) {
	// Support port mapping where a host port[1] is specified in addition to container port
	// [1]: https://cirrus-ci.org/guide/writing-tasks/#additional-containers
	const maxSplits = 2

	portParts := strings.SplitN(port, ":", maxSplits)

	if len(portParts) == maxSplits {
		hostPort, err := strconv.ParseUint(portParts[0], 10, 32)
		if err != nil {
			return nil, err
		}

		containerPort, err := strconv.ParseUint(portParts[1], 10, 32)
		if err != nil {
			return nil, err
		}

		return &api.PortMapping{ContainerPort: uint32(containerPort), HostPort: uint32(hostPort)}, nil
	}

	containerPort, err := strconv.ParseUint(portParts[0], 10, 32)
	if err != nil {
		return nil, err
	}

	return &api.PortMapping{ContainerPort: uint32(containerPort)}, nil
}

// nolint:gocognit
func NewAdditionalContainer(mergedEnv map[string]string, boolevator *boolevator.Boolevator) *AdditionalContainer {
	ac := &AdditionalContainer{
		proto: &api.AdditionalContainer{},
	}

	ac.OptionalField(nameable.NewSimpleNameable("name"), schema.String(""), func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		if name == "main" {
			return node.ParserError("use of reserved name '%s' for an additional container,"+
				" please choose another one", name)
		}

		isNotLetter := func(r rune) bool {
			return !unicode.IsLetter(r)
		}

		if strings.IndexFunc(name, isNotLetter) != -1 {
			return node.ParserError("additional container name '%s' is invalid,"+
				" please only use letters without special symbols", name)
		}

		ac.proto.Name = name

		return nil
	})

	ac.OptionalField(nameable.NewSimpleNameable("environment"), schema.Map(""), func(node *node.Node) error {
		acEnv, err := node.GetMapOrListOfMaps()
		if err != nil {
			return err
		}
		ac.proto.Environment = environment.Merge(ac.proto.Environment, acEnv)
		return nil
	})
	ac.OptionalField(nameable.NewSimpleNameable("env"), schema.Map(""), func(node *node.Node) error {
		acEnv, err := node.GetMapOrListOfMaps()
		if err != nil {
			return err
		}
		ac.proto.Environment = environment.Merge(ac.proto.Environment, acEnv)
		return nil
	})

	imageSchema := schema.String("Docker Image.")
	ac.RequiredField(nameable.NewSimpleNameable("image"), imageSchema, func(node *node.Node) error {
		image, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		ac.proto.Image = image
		return nil
	})

	ac.OptionalField(nameable.NewSimpleNameable("port"), schema.Port(), func(node *node.Node) error {
		port, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		portMapping, err := parsePort(port)
		if err != nil {
			return node.ParserError("failed to parse port: %v", err)
		}

		ac.proto.Ports = append(ac.proto.Ports, portMapping)

		return nil
	})

	ac.OptionalField(nameable.NewSimpleNameable("ports"), schema.Ports(), func(node *node.Node) error {
		ports, err := node.GetSliceOfExpandedStrings(mergedEnv)
		if err != nil {
			return err
		}

		for _, port := range ports {
			portMapping, err := parsePort(port)
			if err != nil {
				return node.ParserError("failed to parse port: %v", err)
			}

			ac.proto.Ports = append(ac.proto.Ports, portMapping)
		}

		return nil
	})

	ac.OptionalField(nameable.NewSimpleNameable("cpu"), schema.Number(""), func(node *node.Node) error {
		cpu, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		cpuFloat, err := strconv.ParseFloat(cpu, 32)
		if err != nil {
			return err
		}
		ac.proto.Cpu = float32(cpuFloat)
		return nil
	})

	ac.OptionalField(nameable.NewSimpleNameable("memory"), schema.Memory(), func(node *node.Node) error {
		memory, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		memoryParsed, err := ParseMegaBytes(memory)
		if err != nil {
			return node.ParserError("%s", err.Error())
		}
		ac.proto.Memory = uint32(memoryParsed)
		return nil
	})

	commandSchema := schema.Script("Container CMD to override.")
	ac.OptionalField(nameable.NewSimpleNameable("command"), commandSchema, func(node *node.Node) error {
		command, err := node.GetSliceOfNonEmptyStrings()
		if err != nil {
			return err
		}

		ac.proto.Command = command

		return nil
	})

	rcommandSchema := schema.Script("Container readiness probe command.")
	ac.OptionalField(nameable.NewSimpleNameable("readiness_command"), rcommandSchema, func(node *node.Node) error {
		readinessCommand, err := node.GetSliceOfNonEmptyStrings()
		if err != nil {
			return err
		}

		ac.proto.ReadinessCommand = readinessCommand

		return nil
	})

	ac.OptionalField(nameable.NewSimpleNameable("privileged"), schema.Condition(""), func(node *node.Node) error {
		privileged, err := node.GetBoolValue(mergedEnv, boolevator)
		if err != nil {
			return err
		}

		ac.proto.Privileged = privileged

		return nil
	})

	return ac
}

func (ac *AdditionalContainer) Parse(node *node.Node) (*api.AdditionalContainer, error) {
	if err := ac.DefaultParser.Parse(node); err != nil {
		return nil, err
	}

	// Once "port" field is deprecated we can mark "ports" field as required and remove this logic
	if len(ac.proto.Ports) == 0 {
		return nil, node.ParserError("should specify either \"port\" or \"ports\"")
	}
	if node.HasChild("port") && node.HasChild("ports") {
		return nil, node.ParserError("please only use \"ports\" field")
	}

	// Resource defaults
	if ac.proto.Cpu == 0 {
		ac.proto.Cpu = defaultAdditionalCPU
	}
	if ac.proto.Memory == 0 {
		ac.proto.Memory = defaultAdditionalMemory
	}

	return ac.proto, nil
}

func (ac *AdditionalContainer) Schema() *jsschema.Schema {
	modifiedSchema := ac.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "Additional Container definition."

	return modifiedSchema
}
