package instance

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/heuristic"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/cirruslabs/echelon"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"io"
	"io/ioutil"
	"math"
	"path"
	"runtime"
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

func NewFromProto(anyInstance *any.Any, commands []*api.Command) (Instance, error) {
	var dynamicInstance ptypes.DynamicAny
	if err := ptypes.UnmarshalAny(anyInstance, &dynamicInstance); err != nil {
		return nil, err
	}

	switch instance := dynamicInstance.Message.(type) {
	case *api.ContainerInstance:
		return &ContainerInstance{
			Image:                instance.Image,
			CPU:                  instance.Cpu,
			Memory:               instance.Memory,
			AdditionalContainers: instance.AdditionalContainers,
		}, nil
	case *api.PipeInstance:
		stages, err := PipeStagesFromCommands(commands)
		if err != nil {
			return nil, err
		}

		return &PipeInstance{
			CPU:    instance.Cpu,
			Memory: instance.Memory,
			Stages: stages,
		}, nil
	case *api.PrebuiltImageInstance:
		// PrebuiltImageInstance is currently missing the domain part to craft the full image name
		// used in the follow-up tasks.
		//
		// However, since currently the only possible value is "gcr.io",
		// we simply craft the image name manually using that hardcoded value.
		image := path.Join("gcr.io", instance.Repository) + ":" + instance.Reference

		return &PrebuiltInstance{
			Image:      image,
			Dockerfile: instance.DockerfilePath,
			Arguments:  instance.Arguments,
		}, nil
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedInstance, instance)
	}
}

type RunConfig struct {
	ProjectDir                 string
	Endpoint                   string
	ServerSecret, ClientSecret string
	TaskID                     int64
	Logger                     *echelon.Logger
	DirtyMode                  bool
	DockerOptions              options.DockerOptions
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

	// Clamp resources to those available for Docker daemon
	info, err := cli.Info(ctx)
	if err != nil {
		return err
	}
	availableCPU := float32(info.NCPU)
	availableMemory := uint32(info.MemTotal / mebi)

	params.CPU = clampCPU(params.CPU, availableCPU)
	params.Memory = clampMemory(params.Memory, availableMemory)
	for _, additionalContainer := range params.AdditionalContainers {
		additionalContainer.Cpu = clampCPU(additionalContainer.Cpu, availableCPU)
		additionalContainer.Memory = clampMemory(additionalContainer.Memory, availableMemory)
	}

	if config.DockerOptions.ShouldPullImage(params.Image) {
		dockerPullLogger := logger.Scoped("docker pull")
		dockerPullLogger.Infof("Pulling image %s...", params.Image)
		progress, err := cli.ImagePull(ctx, params.Image, types.ImagePullOptions{})
		if err != nil {
			dockerPullLogger.Errorf("Failed to pull %s: %v", params.Image, err)
			dockerPullLogger.Finish(false)
			return err
		}
		_, err = io.Copy(ioutil.Discard, progress)
		dockerPullLogger.Finish(err == nil)
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

	if runtime.GOOS == "linux" {
		if heuristic.GetCloudBuildIP(ctx) != "" {
			// Attach the container to the Cloud Build network for RPC the server
			// to be accessible in case we're running in Cloud Build and the CLI
			// itself is containerized (so we can't mount a Unix domain socket
			// because we don't know the path to it on the host)
			hostConfig.NetworkMode = heuristic.CloudBuildNetworkName
		} else {
			// Mount a Unix domain socket in all other Linux cases, assuming that
			// we run in the same mount namespace as the Docker daemon
			socketPath := strings.TrimPrefix(config.Endpoint, "unix://")

			hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: socketPath,
				Target: socketPath,
			})
		}

		// Disable SELinux confinement for this container
		//
		// This solves the following problems when SELinux is enabled:
		// * agent not being able to connect to the CLI's Unix socket
		// * task container not being able to read project directory files when using dirty mode
		hostConfig.SecurityOpt = []string{"label=disable"}
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
		if config.DockerOptions.NoCleanup {
			logger.Infof("not cleaning up container %s, don't forget to remove it with \"docker rm -v %s\"",
				cont.ID, cont.ID)
		} else {
			logger.Debugf("cleaning up container %s", cont.ID)

			err := cli.ContainerRemove(context.Background(), cont.ID, types.ContainerRemoveOptions{
				RemoveVolumes: true,
				Force:         true,
			})
			if err != nil {
				logger.Warnf("error while removing container: %v", err)
			}
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
			if err := runAdditionalContainer(
				additionalContainersCtx,
				logger,
				additionalContainer,
				cli,
				cont.ID,
				config.DockerOptions,
			); err != nil {
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
	dockerOptions options.DockerOptions,
) error {
	logger.Debugf("pulling additional container image %s", additionalContainer.Image)
	progress, err := cli.ImagePull(ctx, additionalContainer.Image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAdditionalContainerFailed, err)
	}
	_, err = io.Copy(ioutil.Discard, progress)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAdditionalContainerFailed, err)
	}

	logger.Debugf("creating additional container")
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
		if dockerOptions.NoCleanup {
			logger.Infof("not cleaning up additional container %s, don't forget to remove it with \"docker rm -v %s\"",
				cont.ID, cont.ID)

			return
		}

		logger.Debugf("cleaning up additional container %s", cont.ID)
		err := cli.ContainerRemove(context.Background(), cont.ID, types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		})
		if err != nil {
			logger.Warnf("Error while removing additional container: %v", err)
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

	logger.Debugf("starting additional container %s", cont.ID)
	if err := cli.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("%w: %v", ErrAdditionalContainerFailed, err)
	}

	logger.Debugf("waiting for additional container %s to finish", cont.ID)
	waitChan, errChan := cli.ContainerWait(ctx, cont.ID, container.WaitConditionNotRunning)
	select {
	case res := <-waitChan:
		logger.Debugf("additional container exited with %v error and exit code %d", res.Error, res.StatusCode)
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

func clampCPU(requested float32, available float32) float32 {
	return float32(math.Min(float64(requested), float64(available)))
}

func clampMemory(requested uint32, available uint32) uint32 {
	if requested > available {
		return available
	}

	return requested
}
