package worker

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
)

type LifecycleSubscriber interface {
	Name() string
	BeforePoll(ctx context.Context, request *api.PollRequest) error
}
