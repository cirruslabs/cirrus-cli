// +build !linux

package containerbackend

import "fmt"

func NewPodman() (ContainerBackend, error) {
	return nil, fmt.Errorf("%w: Podman is only supported on Linux", ErrNewFailed)
}
