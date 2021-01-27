package options

import (
	"context"
	"errors"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
)

type ContainerOptions struct {
	Pull         bool
	NoPullImages []string
	NoCleanup    bool

	DockerfileImageTemplate string
	DockerfileImagePush     bool
}

func (copts ContainerOptions) ShouldPullImage(
	ctx context.Context,
	backend containerbackend.ContainerBackend,
	image string,
) bool {
	for _, noPullImage := range copts.NoPullImages {
		if noPullImage == image {
			return false
		}
	}

	if copts.Pull {
		return true
	}

	return errors.Is(backend.ImageInspect(ctx, image), containerbackend.ErrNotFound)
}
