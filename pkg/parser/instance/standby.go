package instance

import (
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance/isolation"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parserkit"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"time"
)

type StandbyParameters struct {
	proto api.StandbyInstanceParameters

	parseable.DefaultParser
}

func NewStandbyParameters(mergedEnv map[string]string, parserKit *parserkit.ParserKit) *StandbyParameters {
	config := &StandbyParameters{}

	isolationSchema := isolation.NewIsolation(mergedEnv, parserKit).Schema()
	config.OptionalField(nameable.NewSimpleNameable("isolation"), isolationSchema, func(node *node.Node) error {
		isolation := isolation.NewIsolation(mergedEnv, parserKit)

		if err := isolation.Parse(node, parserKit); err != nil {
			return err
		}

		config.proto.Isolation = isolation.Proto()

		return nil
	})

	warmupSchema := NewWarmup(mergedEnv, parserKit).Schema()
	config.OptionalField(nameable.NewSimpleNameable("warmup"), warmupSchema, func(node *node.Node) error {
		warmup := NewWarmup(mergedEnv, parserKit)

		if err := warmup.Parse(node, parserKit); err != nil {
			return err
		}

		config.proto.Warmup = warmup.Proto()

		return nil
	})

	return config
}

func (config *StandbyParameters) Parse(node *node.Node, parserKit *parserkit.ParserKit) error {
	return config.DefaultParser.Parse(node, parserKit)
}

func (config *StandbyParameters) Proto() *api.StandbyInstanceParameters {
	return &config.proto
}

type Warmup struct {
	proto api.StandbyInstanceParameters_Warmup

	parseable.DefaultParser
}

func (warmup *Warmup) Parse(node *node.Node, parserKit *parserkit.ParserKit) error {
	return warmup.DefaultParser.Parse(node, parserKit)
}

func (warmup *Warmup) Proto() *api.StandbyInstanceParameters_Warmup {
	return &warmup.proto
}

func NewWarmup(mergedEnv map[string]string, parserKit *parserkit.ParserKit) *Warmup {
	warmup := &Warmup{}

	scriptSchema := schema.String("Warmup script to run.")
	warmup.OptionalField(nameable.NewSimpleNameable("script"), scriptSchema, func(node *node.Node) error {
		script, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		warmup.proto.Script = script
		return nil
	})

	timeoutSchema := schema.Integer("Warmup timeout.")
	warmup.OptionalField(nameable.NewSimpleNameable("timeout"), timeoutSchema, func(node *node.Node) error {
		timeout, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		timeoutParsed, err := time.ParseDuration(timeout)
		if err != nil {
			return node.ParserError("%s", err.Error())
		}
		warmup.proto.TimeoutSeconds = uint64(timeoutParsed.Seconds())
		return nil
	})

	return warmup
}
