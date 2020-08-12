package instance

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"io"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"sync"
)

var (
	ErrUnsupportedInstance       = errors.New("unsupported instance type")
	ErrAdditionalContainerFailed = errors.New("additional container failed")
)

const (
	mebi = 1024 * 1024
	nano = 1_000_000_000
)

type Instance interface {
	Run(context.Context, *RunConfig) error
}

func NewFromProto(instance *api.Task_Instance, commands []*api.Command) (Instance, error) {
	// Validate and unmarshal the instance descriptor
	switch instance.Type {
	case "container":
		var taskContainer api.ContainerInstance
		if err := proto.Unmarshal(instance.Payload, &taskContainer); err != nil {
			return nil, err
		}
		return &ContainerInstance{
			Image:                taskContainer.Image,
			CPU:                  taskContainer.Cpu,
			Memory:               taskContainer.Memory,
			AdditionalContainers: taskContainer.AdditionalContainers,
		}, nil
	case "pipe":
		var pipe api.PipeInstance
		if err := proto.Unmarshal(instance.Payload, &pipe); err != nil {
			return nil, err
		}

		stages, err := PipeStagesFromCommands(commands)
		if err != nil {
			return nil, err
		}

		return &PipeInstance{
			CPU:    pipe.Cpu,
			Memory: pipe.Memory,
			Stages: stages,
		}, nil
	default:
		return nil, ErrUnsupportedInstance
	}
}

type RunConfig struct {
	ProjectDir                 string
	Endpoint                   string
	ServerSecret, ClientSecret string
	TaskID                     int64
	Logger                     *logrus.Logger
}

type Params struct {
	Image                  string
	CPU                    float32
	Memory                 uint32
	AdditionalContainers   []*api.AdditionalContainer
	CommandFrom, CommandTo string
	WorkingVolumeName      string
}

func RunDockerizedAgent(ctx context.Context, config *RunConfig, params *Params) error {
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

	logger.WithContext(ctx).Debugf("pulling image %s", params.Image)
	progress, err := cli.ImagePull(ctx, params.Image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	_, err = io.Copy(ioutil.Discard, progress)
	if err != nil {
		return err
	}

	logger.WithContext(ctx).Debugf("creating container using working volume %s", params.WorkingVolumeName)
	containerConfig := container.Config{
		Image: params.Image,
		Entrypoint: []string{
			path.Join(WorkingVolumeMountpoint, WorkingVolumeAgent),
			"-api-endpoint",
			config.Endpoint,
			"-server-token",
			config.ServerSecret,
			"-client-token",
			config.ClientSecret,
			"-task-id",
			strconv.FormatInt(config.TaskID, 10),
			"-command-from",
			params.CommandFrom,
			"-command-to",
			params.CommandTo,
		},
	}
	hostConfig := container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: params.WorkingVolumeName,
				Target: WorkingVolumeMountpoint,
			},
		},
		Resources: container.Resources{
			NanoCPUs: int64(params.CPU * nano),
			Memory:   int64(params.Memory * mebi),
		},
	}

	// In case the additional containers are used, tell the agent to wait for them
	if len(params.AdditionalContainers) > 0 {
		var ports []string
		for _, additionalContainer := range params.AdditionalContainers {
			ports = append(ports, strconv.FormatUint(uint64(additionalContainer.ContainerPort), 10))
		}
		commaDelimitedPorts := strings.Join(ports, ",")
		containerConfig.Env = append(containerConfig.Env, "CIRRUS_PORTS_WAIT_FOR="+commaDelimitedPorts)
	}

	cont, err := cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, "")
	if err != nil {
		return err
	}

	// Create controls for the additional containers
	//
	// We also separate the context here to gain a better control of the cancellation order:
	// when the parent context (ctx) is cancelled, the main container will be killed first,
	// and only then all the additional containers will be killed via a separate context
	// (additionalContainersCtx).
	var additionalContainersWG sync.WaitGroup
	additionalContainersCtx, additionalContainersCancel := context.WithCancel(context.Background())

	// Schedule all containers for removal
	defer func() {
		logger.WithContext(ctx).Debugf("cleaning up container %s", cont.ID)
		err := cli.ContainerRemove(context.Background(), cont.ID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			logger.WithContext(ctx).WithError(err).Warn("while removing container")
		}

		additionalContainersCancel()
		additionalContainersWG.Wait()
	}()

	// Start additional containers (if any)
	additionalContainersErrChan := make(chan error, len(params.AdditionalContainers))
	for _, additionalContainer := range params.AdditionalContainers {
		additionalContainer := additionalContainer

		additionalContainersWG.Add(1)
		go func() {
			if err := runAdditionalContainer(additionalContainersCtx, logger, additionalContainer, cli, cont.ID); err != nil {
				additionalContainersErrChan <- err
			}
			additionalContainersWG.Done()
		}()
	}

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
	case acErr := <-additionalContainersErrChan:
		return acErr
	}

	return nil
}

func runAdditionalContainer(
	ctx context.Context,
	logger *logrus.Logger,
	additionalContainer *api.AdditionalContainer,
	cli *client.Client,
	connectToContainer string,
) error {
	logger.WithContext(ctx).Debugf("pulling image %s", additionalContainer.Image)
	progress, err := cli.ImagePull(ctx, additionalContainer.Image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAdditionalContainerFailed, err)
	}
	_, err = io.Copy(ioutil.Discard, progress)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAdditionalContainerFailed, err)
	}

	logger.WithContext(ctx).Debug("creating container")
	containerConfig := container.Config{
		Image: additionalContainer.Image,
		Cmd:   additionalContainer.Command,
		Env:   envMapToSlice(additionalContainer.Environment),
	}
	hostConfig := container.HostConfig{
		Resources: container.Resources{
			NanoCPUs: int64(additionalContainer.Cpu * nano),
			Memory:   int64(additionalContainer.Memory * mebi),
		},
		NetworkMode: container.NetworkMode(fmt.Sprintf("container:%s", connectToContainer)),
	}
	cont, err := cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, "")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAdditionalContainerFailed, err)
	}

	defer func() {
		logger.WithContext(ctx).Debugf("cleaning up container %s", cont.ID)
		err := cli.ContainerRemove(context.Background(), cont.ID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			logger.WithContext(ctx).WithError(err).Warn("while removing container")
		}
	}()

	// We don't support port mappings at this moment: re-implementing them similarly to Kubernetes
	// would require fiddling with Netfilter, which results in unwanted complexity.
	//
	// So here we simply do our best effort and warn the user about potential problems.
	if additionalContainer.HostPort != 0 {
		logger.Warnf("port mappings are unsupported by the Cirrus CLI, please tell the application "+
			"running in the additional container '%s' to use a different port", additionalContainer.Name)
	}

	logger.WithContext(ctx).Debugf("starting container %s", cont.ID)
	if err := cli.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("%w: %v", ErrAdditionalContainerFailed, err)
	}

	logger.WithContext(ctx).Debugf("waiting for container %s to finish", cont.ID)
	waitChan, errChan := cli.ContainerWait(ctx, cont.ID, container.WaitConditionNotRunning)
	select {
	case res := <-waitChan:
		logger.WithContext(ctx).Debugf("container exited with %v error and exit code %d", res.Error, res.StatusCode)
	case err := <-errChan:
		return fmt.Errorf("%w: %v", ErrAdditionalContainerFailed, err)
	}

	return nil
}

func envMapToSlice(envMap map[string]string) (envSlice []string) {
	for envKey, envValue := range envMap {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", envKey, envValue))
	}

	return
}
