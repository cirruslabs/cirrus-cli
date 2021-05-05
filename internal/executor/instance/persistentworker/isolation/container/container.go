package container

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/container"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/pwdir"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"os"
)

type Container struct {
	instance *container.Instance
	tempDir  string
	cleanup  func() error
}

func New(image string, cpu float32, memory uint32) (*Container, error) {
	// Create a working directory that will be used if none was supplied when instantiating from the worker
	tempDir, err := pwdir.StaticTempDirWithDynamicFallback()
	if err != nil {
		return nil, err
	}

	return &Container{
		instance: &container.Instance{
			Image:    image,
			CPU:      cpu,
			Memory:   memory,
			Platform: platform.Auto(),
		},
		tempDir: tempDir,
		cleanup: func() error {
			return os.RemoveAll(tempDir)
		},
	}, nil
}

func (cont *Container) Run(ctx context.Context, config *runconfig.RunConfig) (err error) {
	if config.ProjectDir == "" {
		config.ProjectDir = cont.tempDir
	}

	return cont.instance.Run(ctx, config)
}

func (cont *Container) WorkingDirectory(projectDir string, dirtyMode bool) string {
	return cont.instance.WorkingDirectory(projectDir, dirtyMode)
}

func (cont *Container) Close() error {
	return cont.cleanup()
}
