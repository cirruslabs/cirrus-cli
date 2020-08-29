package logs

import (
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
)

func NewTravisCILogsRenderer(renderer *renderers.SimpleRenderer) echelon.LogRendered {
	return &FoldableLogsRenderer{
		delegate:          renderer,
		startFoldTemplate: "travis_fold:start:%s",
		endFoldTemplate:   "travis_fold:end:%s",
	}
}
