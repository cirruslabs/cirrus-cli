package logs

import (
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
)

type GithubActionsLogsRenderer struct {
	*FoldableLogsRenderer
}

func NewGithubActionsLogsRenderer(renderer *renderers.SimpleRenderer) echelon.LogRendered {
	return &GithubActionsLogsRenderer{
		&FoldableLogsRenderer{
			delegate:          renderer,
			startFoldTemplate: "##[group]%s",
			endFoldTemplate:   "##[endgroup]",
		},
	}
}
