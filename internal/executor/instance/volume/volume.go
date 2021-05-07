package volume

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/cirruslabs/cirrus-cli/internal/executor/pullhelper"
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
	config *runconfig.RunConfig,
	platform platform.Platform,
) (*Volume, *Volume, error) {
	initLogger := config.Logger().Scoped("Preparing execution environment...")
	initLogger.Infof("Preparing volume to work with...")

	identifier := uuid.New().String()
	agentVolumeName := fmt.Sprintf("cirrus-agent-volume-%s", identifier)
	workingVolumeName := fmt.Sprintf("cirrus-working-volume-%s", identifier)

	agentVolume, workingVolume, err := CreateWorkingVolume(ctx, config.ContainerBackend, config.ContainerOptions,
		agentVolumeName, workingVolumeName, config.ProjectDir, config.DirtyMode, config.GetAgentVersion(), platform)
	if err != nil {
		initLogger.Warnf("Failed to create a volume from working directory: %v", err)
		initLogger.Finish(false)

		return nil, nil, err
	}

	initLogger.Finish(true)

	return agentVolume, workingVolume, err
}

// CreateWorkingVolume returns name of the working volume created according to the specification in arguments.
func CreateWorkingVolume(
	ctx context.Context,
	backend containerbackend.ContainerBackend,
	containerOptions options.ContainerOptions,
	agentVolumeName string,
	workingVolumeName string,
	projectDir string,
	dontPopulate bool,
	agentVersion string,
	platform platform.Platform,
) (agentVolume *Volume, vol *Volume, err error) {
	agentImage := platform.ContainerAgentImage(agentVersion)

	if err := pullhelper.PullHelper(ctx, agentImage, backend, containerOptions, nil); err != nil {
		return nil, nil, fmt.Errorf("%w: when pulling agent image: %v", ErrVolumeCreationFailed, err)
	}

	if err := backend.VolumeCreate(ctx, agentVolumeName); err != nil {
		return nil, nil, fmt.Errorf("%w: when creating agent volume: %v", ErrVolumeCreationFailed, err)
	}
	if err := backend.VolumeCreate(ctx, workingVolumeName); err != nil {
		return nil, nil, fmt.Errorf("%w: when creating working volume: %v", ErrVolumeCreationFailed, err)
	}
	defer func() {
		if err != nil {
			_ = backend.VolumeDelete(ctx, agentVolumeName)
			_ = backend.VolumeDelete(ctx, workingVolumeName)
		}
	}()

	copyCommand := platform.ContainerCopyCommand(!dontPopulate)

	// Create and start a helper container that will copy the project directory (if needed) and the agent
	// into the working volume
	input := &containerbackend.ContainerCreateInput{
		Image:   agentImage,
		Command: copyCommand.Command,
		Mounts: []containerbackend.ContainerMount{
			{
				Type:   containerbackend.MountTypeVolume,
				Source: agentVolumeName,
				Target: copyCommand.CopiesAgentToDir,
			},
		},
	}

	// When using non-dirty mode we need to do a full copy of the project directory
	if !dontPopulate {
		input.Mounts = append(input.Mounts, containerbackend.ContainerMount{
			Type:     containerbackend.MountTypeBind,
			Source:   projectDir,
			Target:   copyCommand.CopiesProjectFromDir,
			ReadOnly: true,
		})

		input.Mounts = append(input.Mounts, containerbackend.ContainerMount{
			Type:   containerbackend.MountTypeVolume,
			Source: workingVolumeName,
			Target: copyCommand.CopiesProjectToDir,
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
		return nil, nil, fmt.Errorf("%w: when creating helper container: %v", ErrVolumeCreationFailed, err)
	}
	defer func() {
		removeErr := backend.ContainerDelete(ctx, cont.ID)
		if removeErr != nil {
			err = fmt.Errorf("%w: %v", ErrVolumeCreationFailed, removeErr)
		}
	}()

	err = backend.ContainerStart(ctx, cont.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: when starting helper container: %v", ErrVolumeCreationFailed, err)
	}

	// Wait for the container to finish copying
	waitChan, errChan := backend.ContainerWait(ctx, cont.ID)
	select {
	case res := <-waitChan:
		if res.StatusCode != 0 {
			return nil, nil, fmt.Errorf("%w: helper container exited with %v error and exit code %d",
				ErrVolumeCreationFailed, res.Error, res.StatusCode)
		}
	case err := <-errChan:
		return nil, nil, fmt.Errorf("%w: while waiting for helper container: %v", ErrVolumeCreationFailed, err)
	}

	return &Volume{agentVolumeName}, &Volume{workingVolumeName}, nil
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
