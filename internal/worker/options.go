package worker

import (
	"github.com/cirruslabs/cirrus-cli/internal/worker/security"
	"github.com/cirruslabs/cirrus-cli/internal/worker/standby"
	"github.com/cirruslabs/cirrus-cli/internal/worker/upstream"
	"github.com/sirupsen/logrus"
)

type Option func(*Worker)

func WithLogger(logger logrus.FieldLogger) Option {
	return func(e *Worker) {
		e.logger = logger
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

func WithUpstream(upstream *upstream.Upstream) Option {
	return func(e *Worker) {
		e.upstreams = append(e.upstreams, upstream)
	}
}

func WithSecurity(security *security.Security) Option {
	return func(e *Worker) {
		e.security = security
	}
}

func WithStandby(standby *standby.Standby) Option {
	return func(e *Worker) {
		e.standby = standby
	}
}
