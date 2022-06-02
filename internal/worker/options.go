package worker

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/endpoint"
	"github.com/sirupsen/logrus"
)

type Option func(*Worker)

func WithLogger(logger logrus.FieldLogger) Option {
	return func(e *Worker) {
		e.logger = logger
	}
}

func WithName(name string) Option {
	return func(e *Worker) {
		e.name = name
	}
}

func WithRegistrationToken(registrationToken string) Option {
	return func(e *Worker) {
		e.registrationToken = registrationToken
	}
}

func WithLabels(labels map[string]string) Option {
	return func(e *Worker) {
		e.userSpecifiedLabels = labels
	}
}

func WithResources(resources map[string]float64) Option {
	return func(e *Worker) {
		e.userSpecifiedResources = resources
	}
}

func WithRPCEndpoint(rpcEndpoint string) Option {
	return func(e *Worker) {
		e.rpcEndpoint = rpcEndpoint
	}
}

func WithAgentEndpoint(agentEndpoint endpoint.Endpoint) Option {
	return func(e *Worker) {
		e.agentEndpoint = agentEndpoint
	}
}
