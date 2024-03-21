package worker

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance/isolation"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"gopkg.in/yaml.v3"
	"strconv"
)

type StandbyConfig struct {
	Isolation *api.Isolation
	Resources map[string]float64
}

var ErrIsolationMissing = errors.New("isolation configuration is required for standby")
var ErrUnsupportedIsolation = errors.New("only Tart instances are currently supported for standby")

func (standby *StandbyConfig) UnmarshalYAML(value *yaml.Node) error {
	node, err := node.NewFromNodeWithMergeExemptions(yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			value,
		},
	}, nil)
	if err != nil {
		return err
	}

	isolationNode := node.FindChild("isolation")
	if isolationNode == nil {
		return ErrIsolationMissing
	}
	// Parse isolation
	isolationParser := isolation.NewIsolation(nil, nil)
	if err := isolationParser.Parse(isolationNode, nil); err != nil {
		return err
	}

	if _, ok := isolationParser.Proto().Type.(*api.Isolation_Tart_); !ok {
		return fmt.Errorf("%w, got %T", ErrUnsupportedIsolation, isolationParser.Proto().Type)
	}

	standby.Isolation = isolationParser.Proto()

	// Parse resources
	standby.Resources = make(map[string]float64)
	if resourcesNode := node.FindChild("resources"); resourcesNode != nil {
		for _, resourceNode := range resourcesNode.Children {
			resourceValueRaw, err := resourceNode.FlattenedValue()
			if err != nil {
				return err
			}
			resourceValue, err := strconv.ParseFloat(resourceValueRaw, 64)
			if err != nil {
				return err
			}
			standby.Resources[resourceNode.Name] = resourceValue
		}
	}

	return nil
}
