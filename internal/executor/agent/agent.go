package agent

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"io"
	"io/ioutil"
	"time"
)

const (
	// defaultAgentImage is the image we'll extract the agent from.
	defaultAgentImage = "gcr.io/cirrus-ci-community/cirrus-ci-agent:v1.1"

	// defaultAgentImagePath is a path to an agent's binary in the defaultAgentImage.
	defaultAgentImagePath = "/bin/cirrus-ci-agent"

	// DefaultAgentVolumePath is a path to an agent's binary in the volume that GetAgentVolume creates.
	DefaultAgentVolumePath = "/cirrus-ci-agent"
)

// GetAgentVolume returns a Docker volume name with the agent residing at the DefaultAgentVolumePath inside of it.
//
// A special helper container is used to extract the agent from the image into a volume. This container's name is
// updated afterwards to indicate a successful completion and skip unnecessary steps in the future invocations.
func GetAgentVolume(ctx context.Context) (string, error) {
	// Create a Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", err
	}
	defer cli.Close()

	// Retrieve the latest agent image
	pullResult, err := cli.ImagePull(ctx, defaultAgentImage, types.ImagePullOptions{})
	if err != nil {
		return "", err
	}
	_, err = io.Copy(ioutil.Discard, pullResult)
	if err != nil {
		return "", err
	}

	// Retrieve a short ID for this image
	image, _, err := cli.ImageInspectWithRaw(ctx, defaultAgentImage)
	if err != nil {
		return "", err
	}
	shortImageID, err := truncateDigest(image.ID)
	if err != nil {
		return "", err
	}

	// Craft the name for a final volume and a helper container (which has a temporary and a final stage)
	volumeName := fmt.Sprintf("cirrus-agent-%s", shortImageID)
	containerNameFinal := fmt.Sprintf("cirrus-agent-extractor-%s", shortImageID)
	containerNameTemporary := containerNameFinal + "-tmp"

	// Check the presence of a (1) finished container and (2) a volume, which means that we've actually processed
	// the image already and created a volume named volumeName
	volumeExists, finalContainerExists := true, true

	_, err = cli.VolumeInspect(ctx, volumeName)
	if err != nil {
		if client.IsErrNotFound(err) {
			volumeExists = false
		} else {
			return "", err
		}
	}

	_, err = cli.ContainerInspect(ctx, containerNameFinal)
	if err != nil {
		if client.IsErrNotFound(err) {
			finalContainerExists = false
		} else {
			return "", err
		}
	}

	// Perhaps we can tell the user to use an already created volume
	if volumeExists && finalContainerExists {
		return volumeName, nil
	}

	// Nope, let's work on it
	return createAgentVolume(ctx, cli, containerNameTemporary, containerNameFinal, volumeName)
}

func cleanupContainer(ctx context.Context, cli *client.Client, containerName string) error {
	timeout := time.Second
	err := cli.ContainerStop(ctx, containerName, &timeout)
	if err != nil {
		return err
	}

	err = cli.ContainerRemove(ctx, containerName, types.ContainerRemoveOptions{})
	if err != nil {
		return err
	}

	return nil
}

func createAgentVolume(
	ctx context.Context,
	cli *client.Client,
	containerNameTemporary,
	containerNameFinal,
	volumeName string,
) (string, error) {
	// Cleanup objects that we will be creating
	for _, name := range []string{containerNameTemporary, containerNameFinal} {
		err := cleanupContainer(ctx, cli, name)
		if err != nil && !client.IsErrNotFound(err) {
			return "", err
		}
	}
	err := cli.VolumeRemove(ctx, volumeName, false)
	if err != nil && !client.IsErrNotFound(err) {
		return "", err
	}

	// Create a volume that we'll populate with the agent's binary
	_, err = cli.VolumeCreate(ctx, volume.VolumeCreateBody{Name: volumeName})
	if err != nil {
		return "", err
	}

	// Create a helper container with our volume mounted
	containerConfig := &container.Config{
		Image: defaultAgentImage,

		// We run the agent's binary here (since there may be
		// nothing else to run inside the container) and this
		// should work since agent lingers by default without
		// any options supplied.
		Cmd: []string{defaultAgentImagePath},
	}
	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: "/mnt",
			},
		},
	}
	cont, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, containerNameTemporary)
	if err != nil {
		return "", err
	}

	// Run container to copy the agent
	err = cli.ContainerStart(ctx, containerNameTemporary, types.ContainerStartOptions{})
	if err != nil {
		return "", err
	}

	// Do the copy mumbo-jumbo: first, we create an archive and save it in memory
	agentReader, _, err := cli.CopyFromContainer(ctx, containerNameTemporary, defaultAgentImagePath)
	if err != nil {
		return "", err
	}
	defer agentReader.Close()

	agent, err := ioutil.ReadAll(agentReader)
	if err != nil {
		return "", err
	}
	agentBuf := bytes.NewBuffer(agent)

	// Then, we extract the archive into a directory where our volume is mounted
	//
	// Here it's important to leave the "/" at the end of the dstPath argument
	// since that causes tar to throw /mnt out of the final path in the archive.
	err = cli.CopyToContainer(ctx, cont.ID, "/mnt/", agentBuf, types.CopyToContainerOptions{})
	if err != nil {
		return "", err
	}

	// Done copying, stop the container now
	timeout := time.Second
	err = cli.ContainerStop(ctx, containerNameTemporary, &timeout)
	if err != nil {
		return "", err
	}

	// Signify the operation success
	if err := cli.ContainerRename(ctx, containerNameTemporary, containerNameFinal); err != nil {
		return "", err
	}

	return volumeName, nil
}
