package logs

import (
	"fmt"
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

func (renderer *GithubActionsLogsRenderer) RenderAnnotation(annotation *echelon.Annotation) {
	var mappedLevel string

	switch annotation.Level {
	case echelon.AnnotationLevelNotice:
		mappedLevel = "notice"
	case echelon.AnnotationLevelWarning:
		mappedLevel = "warning"
	case echelon.AnnotationLevelError:
		mappedLevel = "error"
	}

	rawMessage := fmt.Sprintf("::%s file=%s,line=%d,endLine=%d,title=%s::%s", mappedLevel,
		annotation.File, annotation.LineStart, annotation.LineEnd, annotation.Title, annotation.Message)
	renderer.FoldableLogsRenderer.RenderRawMessage(rawMessage)
}
