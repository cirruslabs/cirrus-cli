package instance

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
)

type PersistentWorker struct {
	proto *api.PersistentWorkerInstance

	parseable.DefaultParser
}

func NewPersistentWorker() *PersistentWorker {
	persistentWorker := &PersistentWorker{
		proto: &api.PersistentWorkerInstance{},
	}

	return persistentWorker
}

func (persistentWorker *PersistentWorker) Parse(node *node.Node) (*api.PersistentWorkerInstance, error) {
	if err := persistentWorker.DefaultParser.Parse(node); err != nil {
		return nil, err
	}

	return persistentWorker.proto, nil
}
