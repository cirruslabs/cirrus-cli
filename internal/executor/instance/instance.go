package instance

import (
	"context"
	"errors"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/agent"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/golang/protobuf/proto" //nolint:staticcheck // https://github.com/cirruslabs/cirrus-ci-agent/issues/14
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"path"
	"strconv"
)

type Instance struct {
	image string
}

var ErrUnsupportedInstance = errors.New("unsupported instance type")

const (
	// ContainerProjectDir specifies where in the instance container should we bind-mount the projectDir.
	ContainerProjectDir = "/tmp/cirrus-ci/project-dir"

	// agentVolumeMountPoint specifies where in the instance container should we mount the volume with the agent binary.
	agentVolumeMountPoint = "/tmp/cirrus-ci/agent-dir"
)

func NewFromProto(instance *api.Task_Instance) (*Instance, error) {
	// Validate and unmarshal the instance descriptor
	if instance.Type != "container" {
		return nil, ErrUnsupportedInstance
	}
	var taskContainer api.ContainerInstance
	if err := proto.Unmarshal(instance.Payload, &taskContainer); err != nil {
		return nil, err
	}

	return &Instance{
		image: taskContainer.Image,
	}, nil
}

type RunConfig struct {
	ProjectDir                 string
	Endpoint                   string
	ServerSecret, ClientSecret string
	TaskID                     int64
	Logger                     *logrus.Logger
}

func (inst *Instance) Run(ctx context.Context, config *RunConfig) error {
	logger := config.Logger
	if logger == nil {
		logger = logrus.New()
		logger.Out = ioutil.Discard
	}

	logger.WithContext(ctx).Debug("creating Docker client")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	logger.WithContext(ctx).Debugf("pulling image %s", inst.image)
	progress, err := cli.ImagePull(ctx, inst.image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	_, err = io.Copy(ioutil.Discard, progress)
	if err != nil {
		return err
	}

	agentVolumeName, err := agent.GetAgentVolume(ctx)
	if err != nil {
		return err
	}
	logger.WithContext(ctx).Debugf("using agent from volume %s", agentVolumeName)

	logger.WithContext(ctx).Debug("creating container")
	containerConfig := container.Config{
		Image: inst.image,
		Cmd: []string{
			path.Join(agentVolumeMountPoint, agent.DefaultAgentVolumePath),
			"-api-endpoint",
			config.Endpoint,
			"-insecure-endpoint",
			"-server-token",
			config.ServerSecret,
			"-client-token",
			config.ClientSecret,
			"-task-id",
			strconv.FormatInt(config.TaskID, 10),
		},
	}
	hostConfig := container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeVolume,
				Source:   agentVolumeName,
				Target:   agentVolumeMountPoint,
				ReadOnly: true,
			},
			{
				Type:     mount.TypeBind,
				Source:   config.ProjectDir,
				Target:   ContainerProjectDir,
				ReadOnly: true,
			},
		},
	}
	cont, err := cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, "")
	if err != nil {
		return err
	}

	defer func() {
		logger.WithContext(ctx).Debugf("cleaning up container %s", cont.ID)
		err := cli.ContainerRemove(context.Background(), cont.ID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			logger.WithContext(ctx).WithError(err).Warn("while removing container")
		}
	}()

	logger.WithContext(ctx).Debugf("starting container %s", cont.ID)
	if err := cli.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	logger.WithContext(ctx).Debugf("waiting for container %s to finish", cont.ID)
	waitChan, errChan := cli.ContainerWait(ctx, cont.ID, container.WaitConditionNotRunning)
	select {
	case res := <-waitChan:
		logger.WithContext(ctx).Debugf("container exited with %v error and exit code %d", res.Error, res.StatusCode)
	case err := <-errChan:
		return err
	}

	return nil
}
