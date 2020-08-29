package logs

import (
	"github.com/cirruslabs/echelon"
)

// Foldable log renderer prints start and end messages when a scope is started and finished respectevly
type FoldableLogsRenderer struct {
	delegate          echelon.LogRendered
	startFoldTemplate string
	endFoldTemplate   string
}

func (r FoldableLogsRenderer) RenderScopeStarted(entry *echelon.LogScopeStarted) {
	r.printFoldMessage(entry.GetScopes(), r.startFoldTemplate)
	r.delegate.RenderScopeStarted(entry)
}

func (r FoldableLogsRenderer) RenderScopeFinished(entry *echelon.LogScopeFinished) {
	r.delegate.RenderScopeFinished(entry)
	r.printFoldMessage(entry.GetScopes(), r.endFoldTemplate)
}

func (r FoldableLogsRenderer) RenderMessage(entry *echelon.LogEntryMessage) {
	r.delegate.RenderMessage(entry)
}

func (r FoldableLogsRenderer) printFoldMessage(scopes []string, template string) {
	scopesCount := len(scopes)
	if scopesCount > 0 {
		lastScope := scopes[scopesCount-1]
		foldingMessage := echelon.NewLogEntryMessage(scopes, echelon.InfoLevel, template, lastScope)
		r.delegate.RenderMessage(foldingMessage)
	}
}
