package volume

import (
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"strings"
)

const (
	sourceSplitIdx = iota
	targetSplitIdx
	flagsSplitIdx
)

const (
	numSourceAndTargetSplits      = 2
	numSourceTargetAndFlagsSplits = 3
)

func ParseVolume(node *node.Node, volume string) (*api.Volume, error) {
	splits := strings.Split(volume, ":")

	switch len(splits) {
	case numSourceAndTargetSplits:
		// src:dst
		return &api.Volume{
			Source: splits[sourceSplitIdx],
			Target: splits[targetSplitIdx],
		}, nil
	case numSourceTargetAndFlagsSplits:
		// src:dst:ro
		if splits[flagsSplitIdx] != "ro" {
			return nil, node.ParserError("only \"ro\" volume flag is currently supported")
		}
		return &api.Volume{
			Source:   splits[sourceSplitIdx],
			Target:   splits[targetSplitIdx],
			ReadOnly: true,
		}, nil
	default:
		return nil, node.ParserError("only source:target[:ro] volume specification is currently supported")
	}
}
