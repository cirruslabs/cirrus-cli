package instance

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/google/uuid"
	"runtime"
)

var (
	ErrVolumeCreationFailed = errors.New("working volume creation failed")
	ErrVolumeCleanupFailed  = errors.New("failed to clean up working volume")
)

type Volume struct {
	name string
}

// CreateWorkingVolumeFromConfig returns name of the working volume created according to the specification in config.
func CreateWorkingVolumeFromConfig(
	ctx context.Context,
	config *RunConfig,
	platform platform.Platform,
) (*Volume, error) {
	initLogger := config.Logger.Scoped("Preparing execution environment...")
	initLogger.Infof("Preparing volume to work with...")
	desiredVolumeName := fmt.Sprintf("cirrus-working-volume-%s", uuid.New().String())
	v, err := CreateWorkingVolume(ctx, config.ContainerBackend, desiredVolumeName,
		config.ProjectDir, config.DirtyMode, config.GetAgentVersion(), platform)
	if err != nil {
		initLogger.Warnf("Failed to create a volume from working directory: %v", err)
		initLogger.Finish(false)
		return nil, err
	}
	initLogger.Finish(true)
	return v, err
}

// CreateWorkingVolume returns name of the working volume created according to the specification in arguments.
func CreateWorkingVolume(
	ctx context.Context,
	backend containerbackend.ContainerBackend,
	name string,
	projectDir string,
	dontPopulate bool,
	agentVersion string,
	platform platform.Platform,
) (vol *Volume, err error) {
	agentImage := platform.AgentImage(agentVersion)

	// Retrieve the latest agent image
	if err := backend.ImagePull(ctx, agentImage); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVolumeCreationFailed, err)
	}

	if err := backend.VolumeCreate(ctx, name); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVolumeCreationFailed, err)
	}
	defer func() {
		if err != nil {
			_ = backend.VolumeDelete(ctx, name)
		}
	}()

	// Create and start a helper container that will copy the project directory (if needed) and the agent
	// into the working volume
	input := &containerbackend.ContainerCreateInput{
		Image:   agentImage,
		Command: platform.CopyCommand(!dontPopulate),
		Mounts: []containerbackend.ContainerMount{
			{
				Type:   containerbackend.MountTypeVolume,
				Source: name,
				Target: platform.WorkingVolumeMountpoint(),
			},
		},
	}

	if !dontPopulate {
		input.Mounts = append(input.Mounts, containerbackend.ContainerMount{
			Type:     containerbackend.MountTypeBind,
			Source:   projectDir,
			Target:   platform.ProjectDirMountpoint(),
			ReadOnly: true,
		})

		if runtime.GOOS == "linux" {
			// Disable SELinux confinement for this container, otherwise
			// the rsync might fail when accessing the project directory
			input.DisableSELinux = true
		}
	}

	containerName := fmt.Sprintf("cirrus-helper-container-%s", uuid.New().String())
	cont, err := backend.ContainerCreate(ctx, input, containerName)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVolumeCreationFailed, err)
	}
	defer func() {
		removeErr := backend.ContainerDelete(ctx, cont.ID)
		if removeErr != nil {
			err = fmt.Errorf("%w: %v", ErrVolumeCreationFailed, removeErr)
		}
	}()

	err = backend.ContainerStart(ctx, cont.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVolumeCreationFailed, err)
	}

	// Wait for the container to finish copying
	waitChan, errChan := backend.ContainerWait(ctx, cont.ID)
	select {
	case res := <-waitChan:
		if res.StatusCode != 0 {
			return nil, fmt.Errorf("%w: container exited with %v error and exit code %d",
				ErrVolumeCreationFailed, res.Error, res.StatusCode)
		}
	case err := <-errChan:
		return nil, fmt.Errorf("%w: %v", ErrVolumeCreationFailed, err)
	}

	return &Volume{
		name: name,
	}, nil
}

func (volume *Volume) Name() string {
	return volume.name
}

func (volume *Volume) Close(backend containerbackend.ContainerBackend) error {
	if err := backend.VolumeDelete(context.Background(), volume.name); err != nil {
		return fmt.Errorf("%w: %v", ErrVolumeCleanupFailed, err)
	}

	return nil
}
