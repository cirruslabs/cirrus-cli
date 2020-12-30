package instance

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/heuristic"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/cirruslabs/echelon"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"math"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var (
	ErrFailedToCreateInstance    = errors.New("failed to create instance")
	ErrUnsupportedInstance       = errors.New("unsupported instance type")
	ErrAdditionalContainerFailed = errors.New("additional container failed")
)

const (
	mebi = 1024 * 1024
	nano = 1_000_000_000
)

func NewFromProto(anyInstance *any.Any, commands []*api.Command) (abstract.Instance, error) {
	var dynamicInstance ptypes.DynamicAny
	if err := ptypes.UnmarshalAny(anyInstance, &dynamicInstance); err != nil {
		return nil, err
	}

	switch instance := dynamicInstance.Message.(type) {
	case *api.ContainerInstance:
		var containerPlatform platform.Platform

		switch instance.Platform {
		case api.Platform_LINUX:
			containerPlatform = platform.NewUnix()
		case api.Platform_WINDOWS:
			containerPlatform = platform.NewWindows(instance.OsVersion)
		default:
			return nil, fmt.Errorf("%w: unsupported container instance platform: %s",
				ErrFailedToCreateInstance, instance.Platform.String())
		}

		return &ContainerInstance{
			Image:                instance.Image,
			CPU:                  instance.Cpu,
			Memory:               instance.Memory,
			AdditionalContainers: instance.AdditionalContainers,
			Platform:             containerPlatform,
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
			Dockerfile: instance.Dockerfile,
			Arguments:  instance.Arguments,
		}, nil
	case *api.PersistentWorkerInstance:
		return persistentworker.New(instance.Isolation)
	case *api.DockerBuilder:
		// Ensures that we're not trying to run e.g. Windows-specific scripts on macOS
		instanceOS := strings.ToLower(instance.Platform.String())
		if runtime.GOOS != instanceOS {
			return nil, fmt.Errorf("%w: cannot run %s Docker Builder instance on this platform",
				ErrFailedToCreateInstance, strings.Title(instanceOS))
		}

		return persistentworker.New(&api.Isolation{
			Type: &api.Isolation_None_{
				None: &api.Isolation_None{},
			},
		})
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedInstance, instance)
	}
}

type Params struct {
	Image                  string
	CPU                    float32
	Memory                 uint32
	AdditionalContainers   []*api.AdditionalContainer
	CommandFrom, CommandTo string
	Platform               platform.Platform
	WorkingVolumeName      string
}

func RunContainerizedAgent(ctx context.Context, config *runconfig.RunConfig, params *Params) error {
	logger := config.Logger
	backend := config.ContainerBackend

	// Clamp resources to those available for container backend daemon
	info, err := backend.SystemInfo(ctx)
	if err != nil {
		return err
	}
	availableCPU := float32(info.TotalCPUs)
	availableMemory := uint32(info.TotalMemoryBytes / mebi)

	params.CPU = clampCPU(params.CPU, availableCPU)
	params.Memory = clampMemory(params.Memory, availableMemory)
	for _, additionalContainer := range params.AdditionalContainers {
		additionalContainer.Cpu = clampCPU(additionalContainer.Cpu, availableCPU)
		additionalContainer.Memory = clampMemory(additionalContainer.Memory, availableMemory)
	}

	if config.ContainerOptions.ShouldPullImage(ctx, backend, params.Image) {
		dockerPullLogger := logger.Scoped("image pull")
		dockerPullLogger.Infof("Pulling image %s...", params.Image)
		if err := backend.ImagePull(ctx, params.Image); err != nil {
			dockerPullLogger.Errorf("Failed to pull %s: %v", params.Image, err)
			dockerPullLogger.Finish(false)
			return err
		}
		dockerPullLogger.Finish(true)
	}

	logger.Debugf("creating container using working volume %s", params.WorkingVolumeName)
	input := containerbackend.ContainerCreateInput{
		Image: params.Image,
		Entrypoint: []string{
			params.Platform.AgentBinaryPath(),
			"-api-endpoint",
			config.ContainerEndpoint,
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
		Env: make(map[string]string),
		Mounts: []containerbackend.ContainerMount{
			{
				Type:   containerbackend.MountTypeVolume,
				Source: params.WorkingVolumeName,
				Target: params.Platform.WorkingVolumeMountpoint(),
			},
		},
		Resources: containerbackend.ContainerResources{
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
			input.Network = heuristic.CloudBuildNetworkName
		}

		// Disable SELinux confinement for this container
		//
		// This solves the following problems when SELinux is enabled:
		// * agent not being able to connect to the CLI's Unix socket
		// * task container not being able to read project directory files when using dirty mode
		input.DisableSELinux = true
	}

	// Mount the  directory with the CLI's Unix domain socket in case it's used,
	// assuming that we run in the same mount namespace as the Docker daemon
	if strings.HasPrefix(config.ContainerEndpoint, "unix:") {
		socketPath := strings.TrimPrefix(config.ContainerEndpoint, "unix:")
		socketDir := filepath.Dir(socketPath)

		input.Mounts = append(input.Mounts, containerbackend.ContainerMount{
			Type:   containerbackend.MountTypeBind,
			Source: socketDir,
			Target: socketDir,
		})
	}

	// In dirty mode we mount the project directory in read-write mode
	if config.DirtyMode {
		input.Mounts = append(input.Mounts, containerbackend.ContainerMount{
			Type:   containerbackend.MountTypeBind,
			Source: config.ProjectDir,
			Target: path.Join(params.Platform.WorkingVolumeMountpoint(), platform.WorkingVolumeWorkingDir),
		})
	}

	// In case the additional containers are used, tell the agent to wait for them
	if len(params.AdditionalContainers) > 0 {
		var ports []string
		for _, additionalContainer := range params.AdditionalContainers {
			ports = append(ports, strconv.FormatUint(uint64(additionalContainer.ContainerPort), 10))
		}
		commaDelimitedPorts := strings.Join(ports, ",")
		input.Env["CIRRUS_PORTS_WAIT_FOR"] = commaDelimitedPorts
	}

	cont, err := backend.ContainerCreate(ctx, &input, "")
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
		// We need to remove additional containers first in order to avoid Podman's
		// "has dependent containers which must be removed before it" error
		additionalContainersCancel()
		additionalContainersWG.Wait()

		if config.ContainerOptions.NoCleanup {
			logger.Infof("not cleaning up container %s, don't forget to remove it with \"docker rm -v %s\"",
				cont.ID, cont.ID)
		} else {
			logger.Debugf("cleaning up container %s", cont.ID)

			err := backend.ContainerDelete(context.Background(), cont.ID)
			if err != nil {
				logger.Warnf("error while removing container: %v", err)
			}
		}
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
				backend,
				cont.ID,
				config.ContainerOptions,
			); err != nil {
				additionalContainersErrChan <- err
			}
			additionalContainersWG.Done()
		}()
	}

	logger.Debugf("starting container %s", cont.ID)
	if err := backend.ContainerStart(ctx, cont.ID); err != nil {
		return err
	}

	logger.Debugf("waiting for container %s to finish", cont.ID)
	waitChan, errChan := backend.ContainerWait(ctx, cont.ID)
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
	backend containerbackend.ContainerBackend,
	connectToContainer string,
	containerOptions options.ContainerOptions,
) error {
	logger.Debugf("pulling additional container image %s", additionalContainer.Image)
	err := backend.ImagePull(ctx, additionalContainer.Image)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAdditionalContainerFailed, err)
	}

	logger.Debugf("creating additional container")
	input := &containerbackend.ContainerCreateInput{
		Image:   additionalContainer.Image,
		Command: additionalContainer.Command,
		Env:     additionalContainer.Environment,
		Resources: containerbackend.ContainerResources{
			NanoCPUs: int64(additionalContainer.Cpu * nano),
			Memory:   int64(additionalContainer.Memory * mebi),
		},
		Network: fmt.Sprintf("container:%s", connectToContainer),
	}
	cont, err := backend.ContainerCreate(ctx, input, "")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAdditionalContainerFailed, err)
	}

	defer func() {
		if containerOptions.NoCleanup {
			logger.Infof("not cleaning up additional container %s, don't forget to remove it with \"docker rm -v %s\"",
				cont.ID, cont.ID)

			return
		}

		logger.Debugf("cleaning up additional container %s", cont.ID)
		err := backend.ContainerDelete(context.Background(), cont.ID)
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
	if err := backend.ContainerStart(ctx, cont.ID); err != nil {
		return fmt.Errorf("%w: %v", ErrAdditionalContainerFailed, err)
	}

	logger.Debugf("waiting for additional container %s to finish", cont.ID)
	waitChan, errChan := backend.ContainerWait(ctx, cont.ID)
	select {
	case res := <-waitChan:
		logger.Debugf("additional container exited with %v error and exit code %d", res.Error, res.StatusCode)
	case err := <-errChan:
		return fmt.Errorf("%w: %v", ErrAdditionalContainerFailed, err)
	}

	return nil
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
