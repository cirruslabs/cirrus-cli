package logs

import (
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
	"strings"
)

func NewTeamCityLogsRenderer(renderer *renderers.SimpleRenderer) echelon.LogRendered {
	replacer := strings.NewReplacer(
		"'", "|'",
		"[", "|[",
		"]", "|]",
		"|", "||",
		"\n", "|n",
	)

	return &FoldableLogsRenderer{
		delegate:          renderer,
		startFoldTemplate: "##teamcity[blockOpened name='%s']",
		endFoldTemplate:   "##teamcity[blockClosed name='%s']",
		escapeFunc:        replacer.Replace,
	}
}
