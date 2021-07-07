package instance

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
)

type UnsupportedInstance struct {
	err error
}

func (si *UnsupportedInstance) Run(context.Context, *runconfig.RunConfig) error {
	return si.err
}

func (si *UnsupportedInstance) WorkingDirectory(projectDir string, dirtyMode bool) string {
	return ""
}

func (si *UnsupportedInstance) Close() error {
	return nil
}
