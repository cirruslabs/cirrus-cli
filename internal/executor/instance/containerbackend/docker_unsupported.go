// +build !linux,!darwin,!windows

package containerbackend

import "fmt"

type Docker struct {}

func NewDocker() (ContainerBackend, error) {
	return nil, fmt.Errorf("%w: Docker is only supported on Linux, macOS and Windows", ErrNewFailed)
}
