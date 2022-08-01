package containerbackend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

const containerLogsChannelSize = 512

var (
	ErrNotFound       = errors.New("not found")
	ErrDone           = errors.New("done")
	ErrNewFailed      = errors.New("failed to create container backend")
	ErrBuildFailed    = errors.New("failed to build image")
	ErrPushFailed     = errors.New("failed to push container")
	ErrNotImplemented = errors.New("unimplemented container backend method")
)

type ContainerBackend interface {
	io.Closer

	ImagePull(ctx context.Context, reference string) error
	ImagePush(ctx context.Context, reference string) error
	ImageBuild(ctx context.Context, tarball io.Reader, input *ImageBuildInput) (<-chan string, <-chan error)
	ImageInspect(ctx context.Context, reference string) error
	ImageDelete(ctx context.Context, reference string) error

	VolumeCreate(ctx context.Context, name string) error
	VolumeInspect(ctx context.Context, name string) error
	VolumeDelete(ctx context.Context, name string) error

	ContainerCreate(ctx context.Context, input *ContainerCreateInput, name string) (*ContainerCreateOutput, error)
	ContainerStart(ctx context.Context, id string) error
	ContainerWait(ctx context.Context, id string) (<-chan ContainerWaitResult, <-chan error)
	ContainerLogs(ctx context.Context, id string) (<-chan string, error)
	ContainerDelete(ctx context.Context, id string) error

	SystemInfo(ctx context.Context) (*SystemInfo, error)
}

type ImageBuildInput struct {
	Tags       []string
	Dockerfile string
	BuildArgs  map[string]string
	Pull       bool
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
	Privileged     bool
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
	Version          string
	TotalCPUs        int64
	TotalMemoryBytes int64
}

type Version struct {
	Version string
}

const (
	BackendTypeAuto   = "auto"
	BackendTypeDocker = "docker"
	BackendTypePodman = "podman"
)

func New(name string) (ContainerBackend, error) {
	if name == BackendTypeAuto {
		if nameFromEnv, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); ok {
			name = nameFromEnv
		}
	}

	switch name {
	case BackendTypeDocker:
		return NewDocker()
	case BackendTypePodman:
		return NewPodman()
	case BackendTypeAuto:
		if backend, err := NewDocker(); err == nil {
			return backend, nil
		}

		if backend, err := NewPodman(); err == nil {
			return backend, nil
		}

		return nil, fmt.Errorf("%w: failed to instantiate all supported container backends"+
			" (tried %q and %q, are these actually installed on the system?)",
			ErrNewFailed, BackendTypeDocker, BackendTypePodman)
	default:
		return nil, fmt.Errorf("%w: unknown container backend name %q", ErrNewFailed, name)
	}
}
