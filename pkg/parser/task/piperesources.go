package task

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"strconv"
)

type PipeResources struct {
	cpu    float32
	memory uint32

	parseable.DefaultParser
}

func NewPipeResources(mergedEnv map[string]string) *PipeResources {
	res := &PipeResources{}

	res.RequiredField(nameable.NewSimpleNameable("cpu"), schema.TodoSchema, func(node *node.Node) error {
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

	res.RequiredField(nameable.NewSimpleNameable("memory"), schema.TodoSchema, func(node *node.Node) error {
		memory, err := node.GetExpandedStringValue(mergedEnv)
		if err != nil {
			return err
		}
		memoryParsed, err := instance.ParseMegaBytes(memory)
		if err != nil {
			return err
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
