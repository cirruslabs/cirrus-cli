package worker

import (
	"github.com/cirruslabs/chacha/pkg/localnetworkhelper"
	"github.com/cirruslabs/cirrus-cli/internal/worker/chacha"
	"github.com/cirruslabs/cirrus-cli/internal/worker/resourcemodifier"
	"github.com/cirruslabs/cirrus-cli/internal/worker/security"
	"github.com/cirruslabs/cirrus-cli/internal/worker/upstream"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
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

func WithStandby(standby *api.StandbyInstanceParameters) Option {
	return func(e *Worker) {
		e.standbyParameters = standby
	}
}

func WithResourceModifiersManager(resourceModifiersManager *resourcemodifier.Manager) Option {
	return func(e *Worker) {
		e.resourceModifierManager = resourceModifiersManager
	}
}

func WithTartPrePull(tartPrePull *TartPrePull) Option {
	return func(e *Worker) {
		e.tartPrePull = tartPrePull
	}
}

func WithChacha(chacha *chacha.Chacha) Option {
	return func(e *Worker) {
		e.chacha = chacha
	}
}

func WithLocalNetworkHelper(localNetworkHelper *localnetworkhelper.LocalNetworkHelper) Option {
	return func(e *Worker) {
		e.localNetworkHelper = localNetworkHelper
	}
}
