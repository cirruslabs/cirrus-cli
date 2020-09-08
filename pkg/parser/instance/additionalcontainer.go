package instance

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"strconv"
)

type AdditionalContainer struct {
	proto *api.AdditionalContainer

	parseable.DefaultParser
}

func NewAdditionalContainer(mergedEnv map[string]string) *AdditionalContainer {
	ac := &AdditionalContainer{
		proto: &api.AdditionalContainer{},
	}

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

	return ac.proto, nil
}
