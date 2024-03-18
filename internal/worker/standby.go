package worker

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance/isolation"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"gopkg.in/yaml.v3"
)

type Standby struct {
	Isolation *api.Isolation
}

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

	if _, ok := isolation.Proto().Type.(*api.Isolation_Tart_); !ok {
		return fmt.Errorf("%w, got %T", ErrUnsupportedIsolation, isolation.Proto().Type)
	}

	standby.Isolation = isolation.Proto()

	return nil
}
