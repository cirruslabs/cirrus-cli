package instance

import (
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
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
		environment, err := node.GetStringMapping()
		if err != nil {
			return err
		}
		ac.proto.Environment = environment
		return nil
	})
	ac.OptionalField(nameable.NewSimpleNameable("env"), schema.TodoSchema, func(node *node.Node) error {
		environment, err := node.GetStringMapping()
		if err != nil {
			return err
		}
		ac.proto.Environment = environment
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
		parsedPort, err := strconv.ParseUint(port, 10, 32)
		if err != nil {
			return err
		}
		ac.proto.ContainerPort = uint32(parsedPort)
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
