package worker

import "github.com/sirupsen/logrus"

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

func WithRPCEndpoint(rpcEndpoint string) Option {
	return func(e *Worker) {
		e.rpcEndpoint = rpcEndpoint
	}
}

func WithAgentDirectRPCEndpoint(rpcEndpoint string) Option {
	return func(e *Worker) {
		e.agentDirectRPCEndpoint = rpcEndpoint
	}
}

func WithAgentContainerRPCEndpoint(rpcEndpoint string) Option {
	return func(e *Worker) {
		e.agentContainerRPCEndpoint = rpcEndpoint
	}
}
