package worker

type Option func(*Worker)

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
