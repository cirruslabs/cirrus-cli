package instance

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance/isolation"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parserkit"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	jsschema "github.com/lestrrat-go/jsschema"
)

type PersistentWorker struct {
	proto api.PersistentWorkerInstance

	parseable.DefaultParser
}

func NewPersistentWorker(mergedEnv map[string]string, parserKit *parserkit.ParserKit) *PersistentWorker {
	pworker := &PersistentWorker{}

	labelsSchema := schema.String("Labels for selection.")
	pworker.OptionalField(nameable.NewSimpleNameable("labels"), labelsSchema, func(node *node.Node) error {
		labels, err := node.GetMapOrListOfMapsWithExpansion(mergedEnv)
		if err != nil {
			return err
		}

		pworker.proto.Labels = labels

		return nil
	})

	resourcesSchema := schema.String("Resources to acquire on the Persistent Worker.")
	pworker.OptionalField(nameable.NewSimpleNameable("resources"), resourcesSchema, func(node *node.Node) error {
		resources, err := node.GetFloat64Mapping(mergedEnv)
		if err != nil {
			return err
		}

		pworker.proto.ResourcesToAcquire = resources

		return nil
	})

	isolationSchema := isolation.NewIsolation(mergedEnv, parserKit).Schema()
	pworker.OptionalField(nameable.NewSimpleNameable("isolation"), isolationSchema, func(node *node.Node) error {
		isolation := isolation.NewIsolation(mergedEnv, parserKit)

		if err := isolation.Parse(node, parserKit); err != nil {
			return err
		}

		pworker.proto.Isolation = isolation.Proto()

		return nil
	})

	return pworker
}

func (pworker *PersistentWorker) Parse(
	node *node.Node,
	parserKit *parserkit.ParserKit,
) (*api.PersistentWorkerInstance, error) {
	if err := pworker.DefaultParser.Parse(node, parserKit); err != nil {
		return nil, err
	}

	return &pworker.proto, nil
}

func (pworker *PersistentWorker) Schema() *jsschema.Schema {
	modifiedSchema := pworker.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "Persistent Worker definition."

	return modifiedSchema
}
