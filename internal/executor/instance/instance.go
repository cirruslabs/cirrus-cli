package instance

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/echelon"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
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
	case "prebuilt_image":
		var prebuilt api.PrebuiltImageInstance
		if err := proto.Unmarshal(instance.Payload, &prebuilt); err != nil {
			return nil, err
		}

		// PrebuiltImageInstance is currently missing the domain part to craft the full image name
		// used in the follow-up tasks.
		//
		// However, since currently the only possible value is "gcr.io",
		// we simply craft the image name manually using that hardcoded value.
		image := path.Join("gcr.io", prebuilt.Repository) + ":" + prebuilt.Reference

		return &PrebuiltInstance{
			Image:      image,
			Dockerfile: prebuilt.DockerfilePath,
			Arguments:  prebuilt.Arguments,
		}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedInstance, instance.Type)
	}
}

type RunConfig struct {
	ProjectDir                 string
	Endpoint                   string
	ServerSecret, ClientSecret string
	TaskID                     int64
	Logger                     *echelon.Logger
	DirtyMode                  bool
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
	logger.Debugf("creating Docker client")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	// Check if the image is missing and pull it if needed
	var needToPull bool

	_, _, err = cli.ImageInspectWithRaw(ctx, params.Image)
	if err != nil {
		if client.IsErrNotFound(err) {
			needToPull = true
		} else {
			return err
		}
	}

	if needToPull {
		logger.Debugf("pulling image %s", params.Image)
		progress, err := cli.ImagePull(ctx, params.Image, types.ImagePullOptions{})
		if err != nil {
			return err
		}
		_, err = io.Copy(ioutil.Discard, progress)
		if err != nil {
			return err
		}
	}

	logger.Debugf("creating container using working volume %s", params.WorkingVolumeName)
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

	// In dirty mode we mount the project directory in read-write mode
	if config.DirtyMode {
		hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: config.ProjectDir,
			Target: path.Join(WorkingVolumeMountpoint, WorkingVolumeWorkingDir),
		})
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
		logger.Debugf("cleaning up container %s", cont.ID)
		err := cli.ContainerRemove(context.Background(), cont.ID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			logger.Warnf("error while removing container: %v", err)
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

	logger.Debugf("starting container %s", cont.ID)
	if err := cli.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	logger.Debugf("waiting for container %s to finish", cont.ID)
	waitChan, errChan := cli.ContainerWait(ctx, cont.ID, container.WaitConditionNotRunning)
	select {
	case res := <-waitChan:
		logger.Debugf("container exited with %v error and exit code %d", res.Error, res.StatusCode)
	case err := <-errChan:
		return err
	case acErr := <-additionalContainersErrChan:
		return acErr
	}

	return nil
}

func runAdditionalContainer(
	ctx context.Context,
	logger *echelon.Logger,
	additionalContainer *api.AdditionalContainer,
	cli *client.Client,
	connectToContainer string,
) error {
	logger.Debugf("pulling image %s", additionalContainer.Image)
	progress, err := cli.ImagePull(ctx, additionalContainer.Image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAdditionalContainerFailed, err)
	}
	_, err = io.Copy(ioutil.Discard, progress)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAdditionalContainerFailed, err)
	}

	logger.Debugf("creating container")
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
		logger.Debugf("cleaning up container %s", cont.ID)
		err := cli.ContainerRemove(context.Background(), cont.ID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			logger.Warnf("Error while removing container: %v", err)
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

	logger.Debugf("starting container %s", cont.ID)
	if err := cli.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("%w: %v", ErrAdditionalContainerFailed, err)
	}

	logger.Debugf("waiting for container %s to finish", cont.ID)
	waitChan, errChan := cli.ContainerWait(ctx, cont.ID, container.WaitConditionNotRunning)
	select {
	case res := <-waitChan:
		logger.Debugf("container exited with %v error and exit code %d", res.Error, res.StatusCode)
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
