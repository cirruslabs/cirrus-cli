package instance

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"io"
	"io/ioutil"
	"path"
)

var (
	ErrVolumeCreationFailed = errors.New("working volume creation failed")
	ErrVolumeCleanupFailed  = errors.New("failed to clean up working volume")
)

const (
	// AgentImage is the image we'll use to create a working volume.
	AgentImage = "gcr.io/cirrus-ci-community/cirrus-ci-agent:v1.8.0"

	// Where working volume is mounted to.
	WorkingVolumeMountpoint = "/tmp/cirrus-ci"

	// Agent binary relative to the WorkingVolumeMountpoint.
	WorkingVolumeAgent = "cirrus-ci-agent"

	// Working directory relative to the WorkingVolumeMountpoint.
	WorkingVolumeWorkingDir = "working-dir"
)

type Volume struct {
	name string
}

// CreateWorkingVolumeFromConfig returns name of the working volume created according to the specification in config.
func CreateWorkingVolumeFromConfig(ctx context.Context, config *RunConfig) (*Volume, error) {
	initLogger := config.Logger.Scoped("Preparing execution environment...")
	initLogger.Infof("Preparing volume to work with...")
	desiredVolumeName := fmt.Sprintf("cirrus-working-volume-%s", uuid.New().String())
	v, err := CreateWorkingVolume(ctx, desiredVolumeName, config.ProjectDir, config.DirtyMode)
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
	name string,
	projectDir string,
	dontPopulate bool,
) (vol *Volume, err error) {
	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVolumeCreationFailed, err)
	}
	defer cli.Close()

	// Retrieve the latest agent image
	pullResult, err := cli.ImagePull(ctx, AgentImage, types.ImagePullOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVolumeCreationFailed, err)
	}
	_, err = io.Copy(ioutil.Discard, pullResult)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVolumeCreationFailed, err)
	}

	dockerVolume, err := cli.VolumeCreate(ctx, volume.VolumeCreateBody{Name: name})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVolumeCreationFailed, err)
	}
	defer func() {
		if err != nil {
			_ = cli.VolumeRemove(ctx, dockerVolume.Name, false)
		}
	}()

	// Where we will mount the project directory for further copying into a working volume
	const projectDirMountpoint = "/project-dir"

	// Create and start a helper container that will copy the agent and project directory
	// (if not specified otherwise) into the working volume
	copyCmd := fmt.Sprintf("cp /bin/cirrus-ci-agent %s", path.Join(WorkingVolumeMountpoint, WorkingVolumeAgent))

	if !dontPopulate {
		copyCmd += fmt.Sprintf(" && rsync -r --filter=':- .gitignore' %s/ %s",
			projectDirMountpoint, path.Join(WorkingVolumeMountpoint, WorkingVolumeWorkingDir))
	}

	containerConfig := &container.Config{
		Image: AgentImage,
		Cmd:   []string{"/bin/sh", "-c", copyCmd},
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: dockerVolume.Name,
				Target: WorkingVolumeMountpoint,
			},
		},
	}

	if !dontPopulate {
		hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   projectDir,
			Target:   projectDirMountpoint,
			ReadOnly: true,
		})

		// Disable SELinux confinement for this container, otherwise
		// the rsync might fail when accessing the project directory
		hostConfig.SecurityOpt = []string{"label=disable"}
	}

	cont, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, "")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVolumeCreationFailed, err)
	}
	err = cli.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVolumeCreationFailed, err)
	}

	// Wait for the container to finish copying
	waitChan, errChan := cli.ContainerWait(ctx, cont.ID, container.WaitConditionNotRunning)
	select {
	case res := <-waitChan:
		if res.StatusCode != 0 {
			return nil, fmt.Errorf("%w: container exited with %v error and exit code %d",
				ErrVolumeCreationFailed, res.Error, res.StatusCode)
		}
	case err := <-errChan:
		return nil, fmt.Errorf("%w: %v", ErrVolumeCreationFailed, err)
	}

	// Remove the helper container
	err = cli.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVolumeCreationFailed, err)
	}

	return &Volume{
		name: dockerVolume.Name,
	}, nil
}

func (volume *Volume) Name() string {
	return volume.name
}

func (volume *Volume) Close() error {
	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("%w: %v", ErrVolumeCleanupFailed, err)
	}
	defer cli.Close()

	err = cli.VolumeRemove(context.Background(), volume.name, false)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrVolumeCleanupFailed, err)
	}

	return nil
}
