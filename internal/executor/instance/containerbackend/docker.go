//go:build linux || darwin || windows

package containerbackend

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend/docker"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type Docker struct {
	cli client.APIClient
}

func NewDocker(hosts ...string) (*Docker, error) {
	// Create Docker client
	config, err := config.Load("")
	if err != nil {
		return nil, err
	}

	clientOptions := flags.NewClientOptions()

	clientOptions.Hosts = hosts

	cli, err := command.NewAPIClientFromFlags(clientOptions, config)
	if err != nil {
		return nil, err
	}

	_, err = cli.Ping(context.Background(), client.PingOptions{})
	if err != nil {
		return nil, err
	}

	return &Docker{
		cli: cli,
	}, nil
}

func (backend *Docker) Close() error {
	return backend.cli.Close()
}

func (backend *Docker) ImagePull(ctx context.Context, reference string, architecture *api.Architecture) error {
	options := &client.ImagePullOptions{}

	if architecture != nil {
		switch *architecture {
		case api.Architecture_AMD64:
			options.Platforms = append(options.Platforms, v1.Platform{
				Architecture: "amd64",
				OS:           "linux",
			})
		case api.Architecture_ARM64:
			options.Platforms = append(options.Platforms, v1.Platform{
				Architecture: "arm64",
				OS:           "linux",
			})
		}
	}

	stream, err := backend.cli.ImagePull(ctx, reference, *options)
	if err != nil {
		return err
	}

	if _, err = io.Copy(io.Discard, stream); err != nil {
		return err
	}

	return nil
}

func (backend *Docker) ImagePush(ctx context.Context, reference string) error {
	auth, err := docker.XRegistryAuthForImage(reference)
	if err != nil {
		return err
	}

	stream, err := backend.cli.ImagePush(ctx, reference, client.ImagePushOptions{
		RegistryAuth: auth,
	})
	if err != nil {
		return err
	}
	defer stream.Close()

	rdr := bufio.NewReader(stream)
	for {
		line, isPrefix, err := rdr.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		if isPrefix {
			return fmt.Errorf("%w: received truncated data", ErrPushFailed)
		}

		var streamEntry struct {
			ErrorDetail struct {
				Message string
			}
		}
		if err := json.Unmarshal(line, &streamEntry); err != nil {
			return err
		}

		if streamEntry.ErrorDetail.Message != "" {
			return fmt.Errorf("%w: %s", ErrPushFailed, streamEntry.ErrorDetail.Message)
		}
	}

	return nil
}

func (backend *Docker) ImageBuild(
	ctx context.Context,
	tarball io.Reader,
	input *ImageBuildInput,
) (<-chan string, <-chan error) {
	logChan := make(chan string)
	errChan := make(chan error)

	go func() {
		// Deal with ImageBuildOptions's BuildArgs field quirks
		// since we don't differentiate between empty and missing
		// option values
		pointyArguments := make(map[string]*string)
		for key, value := range input.BuildArgs {
			valueCopy := value
			pointyArguments[key] = &valueCopy
		}

		buildProgress, err := backend.cli.ImageBuild(ctx, tarball, client.ImageBuildOptions{
			Tags:       input.Tags,
			Dockerfile: input.Dockerfile,
			BuildArgs:  pointyArguments,
			Remove:     true,
			PullParent: input.Pull,
		})
		if err != nil {
			errChan <- err
			return
		}

		unrollStream(buildProgress.Body, logChan, errChan)

		if err := buildProgress.Body.Close(); err != nil {
			errChan <- err
			return
		}

		errChan <- ErrDone
	}()

	return logChan, errChan
}

func (backend *Docker) ImageInspect(ctx context.Context, reference string) error {
	_, err := backend.cli.ImageInspect(ctx, reference)

	if cerrdefs.IsNotFound(err) {
		return ErrNotFound
	}

	return err
}

func (backend *Docker) ImageDelete(ctx context.Context, reference string) error {
	_, err := backend.cli.ImageRemove(ctx, reference, client.ImageRemoveOptions{})

	if cerrdefs.IsNotFound(err) {
		return ErrNotFound
	}

	return err
}

func (backend *Docker) VolumeCreate(ctx context.Context, name string) error {
	_, err := backend.cli.VolumeCreate(ctx, client.VolumeCreateOptions{Name: name})
	return err
}

func (backend *Docker) VolumeInspect(ctx context.Context, name string) error {
	_, err := backend.cli.VolumeInspect(ctx, name, client.VolumeInspectOptions{})

	if cerrdefs.IsNotFound(err) {
		return ErrNotFound
	}

	return err
}

func (backend *Docker) VolumeDelete(ctx context.Context, name string) error {
	_, err := backend.cli.VolumeRemove(ctx, name, client.VolumeRemoveOptions{})

	return err
}

func (backend *Docker) ContainerCreate(
	ctx context.Context,
	input *ContainerCreateInput,
	name string,
) (*ContainerCreateOutput, error) {
	containerConfig := container.Config{
		Image:      input.Image,
		Entrypoint: input.Entrypoint,
		Cmd:        input.Command,
		Env:        envMapToSlice(input.Env),
	}
	hostConfig := container.HostConfig{
		Resources: container.Resources{
			NanoCPUs: input.Resources.NanoCPUs,
			Memory:   input.Resources.Memory,
		},
		NetworkMode: container.NetworkMode(input.Network),
		Privileged:  input.Privileged,
	}

	for _, ourMount := range input.Mounts {
		var dockerType mount.Type

		switch ourMount.Type {
		case MountTypeBind:
			dockerType = mount.TypeBind
		case MountTypeVolume:
			dockerType = mount.TypeVolume
		default:
			continue
		}

		newMount := mount.Mount{
			Type:   dockerType,
			Source: ourMount.Source,
			Target: ourMount.Target,
		}

		hostConfig.Mounts = append(hostConfig.Mounts, newMount)
	}

	if input.DisableSELinux {
		hostConfig.SecurityOpt = []string{"label=disable"}
	}

	var platform *v1.Platform

	if input.Architecture != nil {
		switch *input.Architecture {
		case api.Architecture_AMD64:
			platform = &v1.Platform{
				Architecture: "amd64",
			}
		case api.Architecture_ARM64:
			platform = &v1.Platform{
				Architecture: "arm64",
			}
		}
	}

	cont, err := backend.cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:     &containerConfig,
		HostConfig: &hostConfig,
		Platform:   platform,
		Name:       name,
	})
	if err != nil {
		return nil, err
	}

	return &ContainerCreateOutput{
		ID: cont.ID,
	}, nil
}

func (backend *Docker) ContainerStart(ctx context.Context, id string) error {
	_, err := backend.cli.ContainerStart(ctx, id, client.ContainerStartOptions{})

	return err
}

func (backend *Docker) ContainerWait(ctx context.Context, id string) (<-chan ContainerWaitResult, <-chan error) {
	waitChan := make(chan ContainerWaitResult)
	errChan := make(chan error)

	go func() {
		result := backend.cli.ContainerWait(ctx, id, client.ContainerWaitOptions{
			Condition: container.WaitConditionNotRunning,
		})

		select {
		case resp := <-result.Result:
			result := ContainerWaitResult{
				StatusCode: resp.StatusCode,
			}

			if resp.Error != nil {
				result.Error = resp.Error.Message
			}

			waitChan <- result
		case err := <-result.Error:
			errChan <- err
		}
	}()

	return waitChan, errChan
}

func (backend *Docker) ContainerLogs(ctx context.Context, id string) (<-chan string, error) {
	logChan := make(chan string, containerLogsChannelSize)

	multiplexedStream, err := backend.cli.ContainerLogs(ctx, id, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return nil, err
	}

	pipeReader, pipeWriter := io.Pipe()

	go func() {
		_, _ = stdcopy.StdCopy(pipeWriter, pipeWriter, multiplexedStream)
		_ = pipeWriter.Close()
		_ = multiplexedStream.Close()
	}()

	go func() {
		scanner := bufio.NewScanner(pipeReader)

		for scanner.Scan() {
			logChan <- scanner.Text()
		}

		close(logChan)
	}()

	return logChan, nil
}

func (backend *Docker) ContainerDelete(ctx context.Context, id string) error {
	_, err := backend.cli.ContainerRemove(ctx, id, client.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})

	return err
}

func (backend *Docker) SystemInfo(ctx context.Context) (*SystemInfo, error) {
	result, err := backend.cli.Info(ctx, client.InfoOptions{})
	if err != nil {
		return nil, err
	}

	return &SystemInfo{
		Version:          result.Info.ServerVersion,
		TotalCPUs:        int64(result.Info.NCPU),
		TotalMemoryBytes: result.Info.MemTotal,
	}, nil
}

func envMapToSlice(envMap map[string]string) (envSlice []string) {
	for envKey, envValue := range envMap {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", envKey, envValue))
	}

	return
}
