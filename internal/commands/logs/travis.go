package logs

import (
	"github.com/cirruslabs/echelon"
)

func NewTravisCILogsRenderer(renderer echelon.LogRendered) echelon.LogRendered {
	return &FordableLogsRenderer{
		delegate:          renderer,
		startFoldTemplate: "travis_fold:start:%s",
		endFoldTemplate:   "travis_fold:end:%s",
	}
}
