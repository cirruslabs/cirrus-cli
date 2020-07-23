package agent_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/executor/agent"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

// TestGetAgentVolume ensures that the agent's volume created by GetAgentVolume()
// contains the agent binary at the pre-defined location and it's executable.
func TestGetAgentVolume(t *testing.T) {
	agentVolume, err := agent.GetAgentVolume(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Create a Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	ctx := context.Background()

	// Create a container with the agent's volume mounted
	const mountTo = "/agent-volume"
	containerConfig := &container.Config{
		Image: testutil.FetchedImage(t, "debian:latest"),
		Cmd:   []string{"test", "-x", filepath.Join(mountTo, agent.DefaultAgentVolumePath)},
	}
	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: agentVolume,
				Target: mountTo,
			},
		},
	}
	cont, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	// Run the container to check if the agent's binary exists
	err = cli.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the container to stop
	waitChan, errChan := cli.ContainerWait(ctx, cont.ID, container.WaitConditionNotRunning)
	select {
	case ok := <-waitChan:
		// A successful invocation of "test -x" should return 0
		assert.EqualValues(t, 0, ok.StatusCode)
	case err := <-errChan:
		t.Fatal(err)
	}

	// Remove container
	err = cli.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{})
	if err != nil {
		t.Fatal(err)
	}
}
