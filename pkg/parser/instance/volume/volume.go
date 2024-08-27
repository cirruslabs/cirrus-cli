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

func ParseVolume(node *node.Node, volume string) (error, *api.Volume) {
	splits := strings.Split(volume, ":")

	switch len(splits) {
	case numSourceAndTargetSplits:
		// src:dst
		return nil, &api.Volume{
			Source: splits[sourceSplitIdx],
			Target: splits[targetSplitIdx],
		}
	case numSourceTargetAndFlagsSplits:
		// src:dst:ro
		if splits[flagsSplitIdx] != "ro" {
			return node.ParserError("only \"ro\" volume flag is currently supported"), nil
		}
		return nil, &api.Volume{
			Source:   splits[sourceSplitIdx],
			Target:   splits[targetSplitIdx],
			ReadOnly: true,
		}
	default:
		return node.ParserError("only source:target[:ro] volume specification is currently supported"), nil
	}
}
