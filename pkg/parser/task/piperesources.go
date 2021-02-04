package task

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	jsschema "github.com/lestrrat-go/jsschema"
	"strconv"
)

type PipeResources struct {
	cpu    float32
	memory uint32

	parseable.DefaultParser
}

func NewPipeResources(mergedEnv map[string]string) *PipeResources {
	res := &PipeResources{}

	res.OptionalField(nameable.NewSimpleNameable("cpu"), schema.Number(""), func(node *node.Node) error {
		cpu, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		cpuFloat, err := strconv.ParseFloat(cpu, 32)
		if err != nil {
			return err
		}
		res.cpu = float32(cpuFloat)
		return nil
	})

	res.OptionalField(nameable.NewSimpleNameable("memory"), schema.Memory(), func(node *node.Node) error {
		memory, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		memoryParsed, err := instance.ParseMegaBytes(memory)
		if err != nil {
			return node.ParserError("%s", err.Error())
		}
		res.memory = uint32(memoryParsed)
		return nil
	})

	return res
}

func (res *PipeResources) Parse(node *node.Node) error {
	if err := res.DefaultParser.Parse(node); err != nil {
		return err
	}

	return nil
}

func (res *PipeResources) Schema() *jsschema.Schema {
	modifiedSchema := res.DefaultParser.Schema()

	modifiedSchema.Type = jsschema.PrimitiveTypes{jsschema.ObjectType}
	modifiedSchema.Description = "Pipe resources"

	return modifiedSchema
}
