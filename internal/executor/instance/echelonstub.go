package instance

import "github.com/cirruslabs/echelon"

type RendererStub struct{}

func (*RendererStub) RenderScopeStarted(entry *echelon.LogScopeStarted) {}

func (*RendererStub) RenderScopeFinished(entry *echelon.LogScopeFinished) {}

func (*RendererStub) RenderMessage(entry *echelon.LogEntryMessage) {}
