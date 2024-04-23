package container

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/container"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/pwdir"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"go.opentelemetry.io/otel/attribute"
	"os"
)

type Container struct {
	instance *container.Instance
	tempDir  string
	cleanup  func() error
}

func New(image string, cpu float32, memory uint32, volumes []*api.Volume) (*Container, error) {
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
			Volumes:  volumes,
			Platform: platform.Auto(),
		},
		tempDir: tempDir,
		cleanup: func() error {
			return os.RemoveAll(tempDir)
		},
	}, nil
}

func (cont *Container) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("image", cont.instance.Image),
		attribute.String("instance_type", "container"),
	}
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

func (cont *Container) Close(context.Context) error {
	return cont.cleanup()
}
