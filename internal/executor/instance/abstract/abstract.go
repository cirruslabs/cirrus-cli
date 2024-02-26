package abstract

import (
	"context"
	"errors"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/echelon"
)

type Instance interface {
	Run(ctx context.Context, config *runconfig.RunConfig) error
	WorkingDirectory(projectDir string, dirtyMode bool) string
	Close() error
}

type StandbyCapableInstance interface {
	Pull(ctx context.Context, env map[string]string, logger *echelon.Logger) error
	FQN(ctx context.Context) (string, error)
	CloneConfigureStart(ctx context.Context, config *runconfig.RunConfig) (*CloneAndConfigureResult, error)
	Isolation() *api.Isolation

	Instance
}

type CloneAndConfigureResult struct {
	IP                   string
	PreCreatedWorkingDir string
}

var ErrEmptyFQN = errors.New("got empty FQN")
