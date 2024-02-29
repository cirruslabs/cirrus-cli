package standby

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/tart"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance/isolation"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"gopkg.in/yaml.v3"
)

var ErrUnsupportedIsolation = errors.New("only Tart instances are currently supported for standby")

func (standby *Standby) UnmarshalYAML(value *yaml.Node) error {
	// Parse isolation
	isolation := isolation.NewIsolation(nil, nil)

	node, err := node.NewFromNodeWithMergeExemptions(yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			value,
		},
	}, nil)
	if err != nil {
		return err
	}

	if err := isolation.Parse(node, nil); err != nil {
		return err
	}

	standby.Isolation = isolation.Proto()

	switch iso := isolation.Proto().Type.(type) {
	case *api.Isolation_Tart_:
		standby.GetInstance = func() (abstract.StandbyCapableInstance, error) {
			return tart.NewFromIsolation(iso)
		}
	default:
		return fmt.Errorf("%w, got %T", ErrUnsupportedIsolation, iso)
	}

	return nil
}
