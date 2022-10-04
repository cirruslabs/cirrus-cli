package upstream

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/endpoint"
	"github.com/sirupsen/logrus"
)

type Option func(*Upstream)

func WithRPCEndpoint(rpcEndpoint string) Option {
	return func(upstream *Upstream) {
		upstream.rpcEndpoint = rpcEndpoint
	}
}

func WithAgentEndpoint(agentEndpoint endpoint.Endpoint) Option {
	return func(upstream *Upstream) {
		upstream.agentEndpoint = agentEndpoint
	}
}

func WithLogger(logger logrus.FieldLogger) Option {
	return func(upstream *Upstream) {
		upstream.logger = logger
	}
}
