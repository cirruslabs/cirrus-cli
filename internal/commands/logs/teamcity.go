package logs

import (
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
	"strings"
)

func NewTeamCityLogsRenderer(renderer *renderers.SimpleRenderer) echelon.LogRendered {
	return &FoldableLogsRenderer{
		delegate:          renderer,
		startFoldTemplate: "##teamcity[blockOpened name='%s']",
		endFoldTemplate:   "##teamcity[blockClosed name='%s']",
		escapeFunc: func(s string) string {
			replacer := strings.NewReplacer(
				"'", "|'",
				"[", "|[",
				"]", "|]",
				"|", "||",
				"\n", "|n",
			)

			return replacer.Replace(s)
		},
	}
}
