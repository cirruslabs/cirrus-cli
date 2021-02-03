package yaml

import "gopkg.in/yaml.v3"

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
