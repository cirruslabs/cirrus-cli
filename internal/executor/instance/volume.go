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
	"io"
	"io/ioutil"
	"path"
)

var ErrVolumeCreationFailed = errors.New("working volume creation failed")

const (
	// AgentImage is the image we'll use to create a working volume.
	AgentImage = "gcr.io/cirrus-ci-community/cirrus-ci-agent:v1.3.0"

	// Where working volume is mounted to.
	WorkingVolumeMountpoint = "/tmp/cirrus-ci"

	// Agent binary relative to the WorkingVolumeMountpoint.
	WorkingVolumeAgent = "cirrus-ci-agent"

	// Working directory relative to the WorkingVolumeMountpoint.
	WorkingVolumeWorkingDir = "working-dir"
)

// CreateWorkingVolume returns a Docker volume name with the agent and copied projectDir.
func CreateWorkingVolume(ctx context.Context, cli *client.Client, projectDir string) (string, error) {
	// Retrieve the latest agent image
	pullResult, err := cli.ImagePull(ctx, AgentImage, types.ImagePullOptions{})
	if err != nil {
		return "", err
	}
	_, err = io.Copy(ioutil.Discard, pullResult)
	if err != nil {
		return "", err
	}

	vol, err := cli.VolumeCreate(ctx, volume.VolumeCreateBody{})
	if err != nil {
		return "", err
	}

	// Where we will mount the project directory for further copying into a working volume
	const projectDirMountpoint = "/project-dir"

	// Create and start a helper container that will copy the agent and project directory to the working volume
	containerConfig := &container.Config{
		Image: AgentImage,
		Cmd: []string{
			"/bin/sh", "-c", fmt.Sprintf("cp -rT %s %s && cp /bin/cirrus-ci-agent %s",
				projectDirMountpoint, path.Join(WorkingVolumeMountpoint, WorkingVolumeWorkingDir),
				path.Join(WorkingVolumeMountpoint, WorkingVolumeAgent)),
		},
	}
	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: vol.Name,
				Target: WorkingVolumeMountpoint,
			},
			{
				Type:     mount.TypeBind,
				Source:   projectDir,
				Target:   projectDirMountpoint,
				ReadOnly: true,
			},
		},
	}
	cont, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, "")
	if err != nil {
		return "", err
	}
	err = cli.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", err
	}

	// Wait for the container to finish copying
	waitChan, errChan := cli.ContainerWait(ctx, cont.ID, container.WaitConditionNotRunning)
	select {
	case res := <-waitChan:
		if res.StatusCode != 0 {
			return "", fmt.Errorf("%w: container exited with %v error and exit code %d",
				ErrVolumeCreationFailed, res.Error, res.StatusCode)
		}
	case err := <-errChan:
		return "", err
	}

	// Remove the helper container
	err = cli.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{})
	if err != nil {
		return "", err
	}

	return vol.Name, nil
}
