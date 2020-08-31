package logs

import (
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
)

func NewGithubActionsLogsRenderer(renderer *renderers.SimpleRenderer) echelon.LogRendered {
	return &FoldableLogsRenderer{
		delegate:          renderer,
		startFoldTemplate: "##[group]%s",
		endFoldTemplate:   "##[endgroup]",
	}
}
