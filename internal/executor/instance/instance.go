package instance

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/container"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/tart"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/cirruslabs/cirrus-cli/internal/logger"
	"github.com/golang/protobuf/ptypes/any"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"path"
	"runtime"
	"strings"
)

var (
	ErrFailedToCreateInstance = errors.New("failed to create instance")
	ErrUnsupportedInstance    = errors.New("unsupported instance type")
)

func NewFromProto(
	anyInstance *any.Any,
	commands []*api.Command,
	customWorkingDir string,
	logger logger.Lightweight,
) (abstract.Instance, error) {
	if anyInstance == nil {
		return &UnsupportedInstance{
			err: fmt.Errorf("%w: got nil instance which means it's probably not supported by the CLI",
				ErrUnsupportedInstance),
		}, nil
	}

	dynamicInstance, err := anypb.UnmarshalNew(anyInstance, proto.UnmarshalOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal task's instance: %v",
			ErrFailedToCreateInstance, err)
	}

	switch instance := dynamicInstance.(type) {
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

		return &container.Instance{
			Image:                instance.Image,
			CPU:                  instance.Cpu,
			Memory:               instance.Memory,
			AdditionalContainers: instance.AdditionalContainers,
			Platform:             containerPlatform,
			CustomWorkingDir:     customWorkingDir,
		}, nil
	case *api.PipeInstance:
		stages, err := PipeStagesFromCommands(commands)
		if err != nil {
			return nil, err
		}

		return &PipeInstance{
			CPU:              instance.Cpu,
			Memory:           instance.Memory,
			Stages:           stages,
			CustomWorkingDir: customWorkingDir,
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
		return persistentworker.New(instance.Isolation, logger)
	case *api.DockerBuilder:
		// Ensures that we're not trying to run e.g. Windows-specific scripts on macOS
		instanceOS := strings.ToLower(instance.Platform.String())
		if runtime.GOOS != instanceOS {
			return nil, fmt.Errorf("%w: cannot run %s Docker Builder instance on this platform",
				ErrFailedToCreateInstance, cases.Title(language.AmericanEnglish).String(instanceOS))
		}

		return persistentworker.New(&api.Isolation{
			Type: &api.Isolation_None_{
				None: &api.Isolation_None{},
			},
		}, logger)
	case *api.MacOSInstance:
		return tart.New(instance.Image, instance.User, instance.Password, instance.Cpu, instance.Memory,
			tart.WithLogger(logger))
	default:
		return &UnsupportedInstance{
			err: fmt.Errorf("%w: %T", ErrUnsupportedInstance, instance),
		}, nil
	}
}
