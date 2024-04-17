package worker

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance/isolation"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/issue"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parserkit"
	"gopkg.in/yaml.v3"
	"strconv"
)

type StandbyConfig struct {
	Isolation *api.Isolation
	Resources map[string]float64
}

var ErrIsolationMissing = errors.New("isolation configuration is required for standby")
var ErrUnsupportedIsolation = errors.New("only Tart and Vetu instances are currently supported for standby")

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
	parserKit := &parserkit.ParserKit{
		Boolevator:    boolevator.New(),
		IssueRegistry: issue.NewRegistry(),
	}
	isolationParser := isolation.NewIsolation(nil, parserKit)
	if err := isolationParser.Parse(isolationNode, parserKit); err != nil {
		return err
	}

	// Only allow Tart and Vetu to be configured as standby
	switch isolationType := isolationParser.Proto().Type.(type) {
	case *api.Isolation_Tart_:
		// OK
	case *api.Isolation_Vetu_:
		// OK
	default:
		return fmt.Errorf("%w, got %T", ErrUnsupportedIsolation, isolationType)
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
