package logs

import (
	"github.com/cirruslabs/echelon"
)

type FordableLogsRenderer struct {
	delegate          echelon.LogRendered
	startFoldTemplate string
	endFoldTemplate   string
}

func (r FordableLogsRenderer) RenderScopeStarted(entry *echelon.LogScopeStarted) {
	scopes := entry.GetScopes()
	scopesCount := len(scopes)
	if scopesCount > 0 {
		lastScope := scopes[scopesCount-1]
		echelon.NewLogEntryMessage(scopes, echelon.InfoLevel, r.startFoldTemplate, lastScope)
	}
	r.delegate.RenderScopeStarted(entry)
}

func (r FordableLogsRenderer) RenderScopeFinished(entry *echelon.LogScopeFinished) {
	r.delegate.RenderScopeFinished(entry)
	scopes := entry.GetScopes()
	scopesCount := len(scopes)
	if scopesCount > 0 {
		lastScope := scopes[scopesCount-1]
		echelon.NewLogEntryMessage(scopes, echelon.InfoLevel, r.endFoldTemplate, lastScope)
	}
}

func (r FordableLogsRenderer) RenderMessage(entry *echelon.LogEntryMessage) {
	r.delegate.RenderMessage(entry)
}
