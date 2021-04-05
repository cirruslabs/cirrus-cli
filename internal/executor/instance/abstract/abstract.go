package abstract

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
)

type Instance interface {
	Run(context.Context, *runconfig.RunConfig) error
	WorkingDirectory(projectDir string, dirtyMode bool) string
	Close() error
}
