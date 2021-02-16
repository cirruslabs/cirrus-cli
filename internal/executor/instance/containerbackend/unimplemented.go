package containerbackend

import (
	"context"
	"io"
)

type Unimplemented struct{}

func (*Unimplemented) Close() error { return nil }

func (*Unimplemented) ImagePull(ctx context.Context, reference string) error {
	return ErrNotImplemented
}

func (*Unimplemented) ImagePush(ctx context.Context, reference string) error {
	return ErrNotImplemented
}

func (*Unimplemented) ImageBuild(
	ctx context.Context,
	tarball io.Reader,
	input *ImageBuildInput,
) (<-chan string, <-chan error) {
	logChan := make(chan string)
	errChan := make(chan error)

	go func() {
		errChan <- ErrNotImplemented
	}()

	return logChan, errChan
}

func (*Unimplemented) ImageInspect(ctx context.Context, reference string) error {
	return ErrNotImplemented
}

func (*Unimplemented) ImageDelete(ctx context.Context, reference string) error {
	return ErrNotImplemented
}

func (*Unimplemented) VolumeCreate(ctx context.Context, name string) error { return ErrNotImplemented }

func (*Unimplemented) VolumeInspect(ctx context.Context, name string) error { return ErrNotImplemented }

func (*Unimplemented) VolumeDelete(ctx context.Context, name string) error { return ErrNotImplemented }

func (*Unimplemented) ContainerCreate(
	ctx context.Context,
	input *ContainerCreateInput,
	name string,
) (*ContainerCreateOutput, error) {
	return nil, ErrNotImplemented
}

func (*Unimplemented) ContainerStart(ctx context.Context, id string) error { return ErrNotImplemented }

func (*Unimplemented) ContainerWait(ctx context.Context, id string) (<-chan ContainerWaitResult, <-chan error) {
	waitChan := make(chan ContainerWaitResult)
	errChan := make(chan error)

	go func() {
		errChan <- ErrNotImplemented
	}()

	return waitChan, errChan
}

func (*Unimplemented) ContainerLogs(ctx context.Context, id string) (<-chan string, error) {
	return nil, ErrNotImplemented
}

func (*Unimplemented) ContainerDelete(ctx context.Context, id string) error { return ErrNotImplemented }

func (*Unimplemented) SystemInfo(ctx context.Context) (*SystemInfo, error) {
	return nil, ErrNotImplemented
}
