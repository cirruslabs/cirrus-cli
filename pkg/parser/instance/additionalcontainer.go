package instance

import (
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
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

// nolint:gocognit
func NewAdditionalContainer(mergedEnv map[string]string) *AdditionalContainer {
	ac := &AdditionalContainer{
		proto: &api.AdditionalContainer{},
	}

	ac.OptionalField(nameable.NewSimpleNameable("name"), schema.TodoSchema, func(node *node.Node) error {
		name, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		if name == "main" {
			return fmt.Errorf("%w: use of reserved name '%s' for an additional container, please choose another one",
				parsererror.ErrParsing, name)
		}

		isNotLetter := func(r rune) bool {
			return !unicode.IsLetter(r)
		}

		if strings.IndexFunc(name, isNotLetter) != -1 {
			return fmt.Errorf("%w: additional container name '%s' is invalid, please only use letters without special symbols",
				parsererror.ErrParsing, name)
		}

		ac.proto.Name = name

		return nil
	})

	ac.OptionalField(nameable.NewSimpleNameable("environment"), schema.TodoSchema, func(node *node.Node) error {
		acEnv, err := node.GetStringMapping()
		if err != nil {
			return err
		}
		ac.proto.Environment = environment.Merge(ac.proto.Environment, acEnv)
		return nil
	})
	ac.OptionalField(nameable.NewSimpleNameable("env"), schema.TodoSchema, func(node *node.Node) error {
		acEnv, err := node.GetStringMapping()
		if err != nil {
			return err
		}
		ac.proto.Environment = environment.Merge(ac.proto.Environment, acEnv)
		return nil
	})

	ac.OptionalField(nameable.NewSimpleNameable("image"), schema.TodoSchema, func(node *node.Node) error {
		image, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		ac.proto.Image = image
		return nil
	})

	ac.OptionalField(nameable.NewSimpleNameable("port"), schema.TodoSchema, func(node *node.Node) error {
		port, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}

		// Support port mapping where a host port[1] is specified in addition to container port
		// [1]: https://cirrus-ci.org/guide/writing-tasks/#additional-containers
		const maxSplits = 2

		portParts := strings.SplitN(port, ":", maxSplits)

		if len(portParts) == maxSplits {
			hostPort, err := strconv.ParseUint(portParts[0], 10, 32)
			if err != nil {
				return err
			}
			ac.proto.HostPort = uint32(hostPort)

			containerPort, err := strconv.ParseUint(portParts[1], 10, 32)
			if err != nil {
				return err
			}
			ac.proto.ContainerPort = uint32(containerPort)
		} else {
			containerPort, err := strconv.ParseUint(portParts[0], 10, 32)
			if err != nil {
				return err
			}
			ac.proto.ContainerPort = uint32(containerPort)
		}

		return nil
	})

	ac.OptionalField(nameable.NewSimpleNameable("cpu"), schema.TodoSchema, func(node *node.Node) error {
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

	ac.OptionalField(nameable.NewSimpleNameable("memory"), schema.TodoSchema, func(node *node.Node) error {
		memory, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		memoryParsed, err := ParseMegaBytes(memory)
		if err != nil {
			return err
		}
		ac.proto.Memory = uint32(memoryParsed)
		return nil
	})

	return ac
}

func (ac *AdditionalContainer) Parse(node *node.Node) (*api.AdditionalContainer, error) {
	if err := ac.DefaultParser.Parse(node); err != nil {
		return nil, err
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
