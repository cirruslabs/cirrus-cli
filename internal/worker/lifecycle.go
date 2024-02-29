package worker

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
)

type LifecycleSubscriber interface {
	Name() string
	BeforePoll(ctx context.Context, request *api.PollRequest) error
	BeforeRunInstance(ctx context.Context, inst abstract.Instance) error
}
