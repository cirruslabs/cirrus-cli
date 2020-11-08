package containerbackend

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"io"
	"io/ioutil"
	"strings"
)

type Docker struct {
	cli *client.Client
}

func NewDocker() (ContainerBackend, error) {
	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &Docker{
		cli: cli,
	}, nil
}

func (docker *Docker) Close() error {
	return docker.cli.Close()
}

func (docker *Docker) ImagePull(ctx context.Context, reference string) error {
	stream, err := docker.cli.ImagePull(ctx, reference, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	if _, err = io.Copy(ioutil.Discard, stream); err != nil {
		return err
	}

	return nil
}

func (docker *Docker) ImageBuild(
	ctx context.Context,
	tarball io.Reader,
	input *ImageBuildInput,
) (<-chan string, <-chan error) {
	logChan := make(chan string)
	errChan := make(chan error)

	go func() {
		buildProgress, err := docker.cli.ImageBuild(ctx, tarball, types.ImageBuildOptions{
			Tags:       input.Tags,
			Dockerfile: input.Dockerfile,
			BuildArgs:  input.BuildArgs,
			Remove:     true,
		})
		if err != nil {
			errChan <- err
			return
		}

		buildProgressReader := bufio.NewReader(buildProgress.Body)

		for {
			// Docker build progress is line-based
			line, _, err := buildProgressReader.ReadLine()
			if err != nil {
				if err == io.EOF {
					break
				}

				errChan <- err
				return
			}

			// Each line is a JSON object with the actual message wrapped in it
			msg := &struct {
				Stream string
			}{}
			if err := json.Unmarshal(line, &msg); err != nil {
				errChan <- err
				return
			}

			// We're only interested with messages containing the "stream" field, as these are the most helpful
			if msg.Stream == "" {
				continue
			}

			// Cut the unnecessary formatting done by the Docker daemon for some reason
			progressMessage := strings.TrimSpace(msg.Stream)

			// Some messages contain only "\n", so filter these out
			if progressMessage == "" {
				continue
			}

			logChan <- progressMessage
		}

		if err := buildProgress.Body.Close(); err != nil {
			errChan <- err
			return
		}

		errChan <- ErrDone
	}()

	return logChan, errChan
}

func (docker *Docker) ImageInspect(ctx context.Context, reference string) error {
	_, _, err := docker.cli.ImageInspectWithRaw(ctx, reference)

	if client.IsErrNotFound(err) {
		return ErrNotFound
	}

	return err
}

func (docker *Docker) VolumeCreate(ctx context.Context, name string) error {
	_, err := docker.cli.VolumeCreate(ctx, volume.VolumeCreateBody{Name: name})
	return err
}

func (docker *Docker) VolumeInspect(ctx context.Context, name string) error {
	_, err := docker.cli.VolumeInspect(ctx, name)

	if client.IsErrNotFound(err) {
		return ErrNotFound
	}

	return err
}

func (docker *Docker) VolumeDelete(ctx context.Context, name string) error {
	return docker.cli.VolumeRemove(ctx, name, false)
}

func (docker *Docker) ContainerCreate(
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

	cont, err := docker.cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, name)
	if err != nil {
		return nil, err
	}

	return &ContainerCreateOutput{
		ID: cont.ID,
	}, nil
}

func (docker *Docker) ContainerStart(ctx context.Context, id string) error {
	return docker.cli.ContainerStart(ctx, id, types.ContainerStartOptions{})
}

func (docker *Docker) ContainerWait(ctx context.Context, id string) (<-chan ContainerWaitResult, <-chan error) {
	waitChan := make(chan ContainerWaitResult)
	errChan := make(chan error)

	go func() {
		dockerWaitChan, dockerErrChan := docker.cli.ContainerWait(ctx, id, container.WaitConditionNotRunning)

		select {
		case resp := <-dockerWaitChan:
			result := ContainerWaitResult{
				StatusCode: resp.StatusCode,
			}

			if resp.Error != nil {
				result.Error = resp.Error.Message
			}

			waitChan <- result
		case err := <-dockerErrChan:
			errChan <- err
		}
	}()

	return waitChan, errChan
}

func (docker *Docker) ContainerDelete(ctx context.Context, id string) error {
	return docker.cli.ContainerRemove(ctx, id, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
}

func (docker *Docker) SystemInfo(ctx context.Context) (*SystemInfo, error) {
	info, err := docker.cli.Info(ctx)
	if err != nil {
		return nil, err
	}

	return &SystemInfo{
		TotalCPUs:        int64(info.NCPU),
		TotalMemoryBytes: info.MemTotal,
	}, nil
}

func envMapToSlice(envMap map[string]string) (envSlice []string) {
	for envKey, envValue := range envMap {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", envKey, envValue))
	}

	return
}
