package worker

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/issue"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parserkit"
	"gopkg.in/yaml.v3"
)

type StandbyConfig struct {
	*api.StandbyInstanceParameters
}

var ErrIsolationMissing = errors.New("isolation configuration is required for standby")
var ErrUnsupportedIsolation = errors.New("only Tart and Vetu instances are currently supported for standby")

func (standby *StandbyConfig) UnmarshalYAML(value *yaml.Node) error {
	documentNode, err := node.NewFromNodeWithMergeExemptions(yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			value,
		},
	}, nil)
	if err != nil {
		return err
	}

	if isolationNode := documentNode.FindChild("isolation"); isolationNode == nil {
		return ErrIsolationMissing
	}
	// Parse isolation
	parserKit := &parserkit.ParserKit{
		Boolevator:    boolevator.New(),
		IssueRegistry: issue.NewRegistry(),
	}
	parametersParser := instance.NewStandbyParameters(nil, parserKit)
	if err := parametersParser.Parse(documentNode, parserKit); err != nil {
		return err
	}

	// Only allow Tart and Vetu to be configured as standby
	switch isolationType := parametersParser.Proto().Isolation.Type.(type) {
	case *api.Isolation_Tart_:
		// OK
	case *api.Isolation_Vetu_:
		// OK
	default:
		return fmt.Errorf("%w, got %T", ErrUnsupportedIsolation, isolationType)
	}

	standby.StandbyInstanceParameters = parametersParser.Proto()

	return nil
}
