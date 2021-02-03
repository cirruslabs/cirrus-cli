package yaml

import (
	"gopkg.in/yaml.v3"
	"strings"
)

const DefaultYamlMarshalIndent = 2

func PrettyPrint(node *yaml.Node) (string, error) {
	builder := &strings.Builder{}
	encoder := yaml.NewEncoder(builder)
	encoder.SetIndent(DefaultYamlMarshalIndent)
	err := encoder.Encode(node)
	if err != nil {
		return "", err
	}
	err = encoder.Close()
	if err != nil {
		return "", err
	}

	return builder.String(), nil
}

func NewSeqNode(content []*yaml.Node) *yaml.Node {
	var result yaml.Node
	result.Kind = yaml.SequenceNode
	result.Tag = "!!seq"
	result.Content = content
	return &result
}

func NewMapNode(content []*yaml.Node) *yaml.Node {
	var result yaml.Node
	result.Kind = yaml.MappingNode
	result.Tag = "!!map"
	result.Content = content
	return &result
}

func NewStringNode(text string) *yaml.Node {
	var result yaml.Node
	result.SetString(text)
	return &result
}
