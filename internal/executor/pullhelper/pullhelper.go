package pullhelper

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
)

func PullHelper(
	ctx context.Context,
	reference string,
	backend containerbackend.ContainerBackend,
	copts options.ContainerOptions,
	logger *echelon.Logger,
) error {
	if !copts.ShouldPullImage(ctx, backend, reference) {
		return nil
	}

	if logger == nil {
		logger = echelon.NewLogger(echelon.ErrorLevel, &renderers.StubRenderer{})
	}

	dockerPullLogger := logger.Scoped("image pull")
	dockerPullLogger.Infof("Pulling image %s...", reference)

	if err := backend.ImagePull(ctx, reference); err != nil {
		dockerPullLogger.Errorf("Failed to pull %s: %v", reference, err)
		dockerPullLogger.Finish(false)

		return err
	}

	dockerPullLogger.Finish(true)

	return nil
}
