package containerbackend

import (
	"context"
	"errors"
	"io"
)

var (
	ErrNotFound = errors.New("not found")
	ErrDone     = errors.New("done")
)

type ContainerBackend interface {
	io.Closer

	ImagePull(ctx context.Context, reference string) error
	ImageBuild(ctx context.Context, tarball io.Reader, input *ImageBuildInput) (<-chan string, <-chan error)
	ImageInspect(ctx context.Context, reference string) error

	VolumeCreate(ctx context.Context, name string) error
	VolumeInspect(ctx context.Context, name string) error
	VolumeDelete(ctx context.Context, name string) error

	ContainerCreate(ctx context.Context, input *ContainerCreateInput, name string) (*ContainerCreateOutput, error)
	ContainerStart(ctx context.Context, id string) error
	ContainerWait(ctx context.Context, id string) (<-chan ContainerWaitResult, <-chan error)
	ContainerDelete(ctx context.Context, id string) error

	SystemInfo(ctx context.Context) (*SystemInfo, error)
}

type ImageBuildInput struct {
	Tags       []string
	Dockerfile string
	BuildArgs  map[string]*string
}

type ContainerCreateInput struct {
	Image          string
	Entrypoint     []string
	Command        []string
	Env            map[string]string
	Mounts         []ContainerMount
	Network        string
	Resources      ContainerResources
	DisableSELinux bool
}

type ContainerMountType int

const (
	MountTypeBind ContainerMountType = iota
	MountTypeVolume
)

type ContainerMount struct {
	Type     ContainerMountType
	Source   string
	Target   string
	ReadOnly bool
}

type ContainerResources struct {
	NanoCPUs int64
	Memory   int64
}

type ContainerCreateOutput struct {
	ID string
}

type ContainerWaitResult struct {
	StatusCode int64
	Error      string
}

type SystemInfo struct {
	TotalCPUs        int64
	TotalMemoryBytes int64
}
