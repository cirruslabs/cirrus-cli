//go:build !linux

package containerbackend

import "fmt"

type Podman struct {
	Unimplemented
}

func NewPodman() (ContainerBackend, error) {
	return nil, fmt.Errorf("%w: Podman is only supported on Linux", ErrNewFailed)
}
