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
	"github.com/golang/protobuf/proto" //nolint:staticcheck // https://github.com/cirruslabs/cirrus-ci-agent/issues/14
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"sync"
)

type Instance struct {
	image                string
	cpu                  float32
	memory               uint32
	additionalContainers []*api.AdditionalContainer
}

var (
	ErrUnsupportedInstance       = errors.New("unsupported instance type")
	ErrAdditionalContainerFailed = errors.New("additional container failed")
)

const (
	mebi = 1024 * 1024
	nano = 1_000_000_000
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
		image:                taskContainer.Image,
		cpu:                  taskContainer.Cpu,
		memory:               taskContainer.Memory,
		additionalContainers: taskContainer.AdditionalContainers,
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

	workingVolumeName, err := CreateWorkingVolume(ctx, cli, config.ProjectDir)
	if err != nil {
		return err
	}
	logger.WithContext(ctx).Debugf("using working volume %s", workingVolumeName)

	logger.WithContext(ctx).Debug("creating container")
	containerConfig := container.Config{
		Image: inst.image,
		Cmd: []string{
			path.Join(WorkingVolumeMountpoint, WorkingVolumeAgent),
			"-api-endpoint",
			config.Endpoint,
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
				Type:   mount.TypeVolume,
				Source: workingVolumeName,
				Target: WorkingVolumeMountpoint,
			},
		},
		Resources: container.Resources{
			NanoCPUs: int64(inst.cpu * nano),
			Memory:   int64(inst.memory * mebi),
		},
	}

	// In case the additional containers are used, tell the agent to wait for them
	if len(inst.additionalContainers) > 0 {
		var ports []string
		for _, additionalContainer := range inst.additionalContainers {
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

		logger.WithContext(ctx).Debugf("cleaning up working volume %s", workingVolumeName)
		err = cli.VolumeRemove(context.Background(), workingVolumeName, false)
		if err != nil {
			logger.WithContext(ctx).WithError(err).Warn("while removing working volume")
		}

		additionalContainersCancel()
		additionalContainersWG.Wait()
	}()

	// Start additional containers (if any)
	additionalContainersErrChan := make(chan error, len(inst.additionalContainers))
	for _, additionalContainer := range inst.additionalContainers {
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
