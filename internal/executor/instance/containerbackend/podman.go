package containerbackend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/antihax/optional"
	swagger "github.com/cirruslabs/cirrus-cli/internal/podmanapi"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

var ErrPodman = errors.New("Podman error")

type Podman struct {
	basePath   string
	httpClient *http.Client
	cli        *swagger.APIClient
}

func NewPodman() ContainerBackend {
	podman := &Podman{
		basePath: "http://d/v1.0.0",
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", "/tmp/podman.sock")
				},
			},
		},
	}

	// Create Podman client
	podman.cli = swagger.NewAPIClient(&swagger.Configuration{
		BasePath:   podman.basePath,
		HTTPClient: podman.httpClient,
	})

	return podman
}

func (podman *Podman) Close() error {
	return nil
}

func (podman *Podman) VolumeCreate(ctx context.Context, name string) error {
	// nolint:bodyclose // already closed by Swagger-generated code
	_, _, err := podman.cli.VolumesApi.LibpodCreateVolume(ctx, &swagger.VolumesApiLibpodCreateVolumeOpts{
		Body: optional.NewInterface(swagger.VolumeCreate{
			Name: name,
		}),
	})

	// Enrich the error with it's cause if possible
	if err != nil {
		if cause := swaggerCause(err); cause != "" {
			return fmt.Errorf("%w: caused by %s", err, swaggerCause(err))
		}
	}

	return err
}

func (podman *Podman) VolumeInspect(ctx context.Context, name string) error {
	// nolint:bodyclose // already closed by Swagger-generated code
	_, resp, err := podman.cli.VolumesApi.LibpodInspectVolume(ctx, name)

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	// Enrich the error with it's cause if possible
	if err != nil {
		if cause := swaggerCause(err); cause != "" {
			return fmt.Errorf("%w: caused by %s", err, swaggerCause(err))
		}
	}

	return err
}

func (podman *Podman) VolumeDelete(ctx context.Context, name string) error {
	// nolint:bodyclose // already closed by Swagger-generated code
	_, err := podman.cli.VolumesApi.LibpodRemoveVolume(ctx, name, &swagger.VolumesApiLibpodRemoveVolumeOpts{
		Force: optional.NewBool(false),
	})

	// Enrich the error with it's cause if possible
	if err != nil {
		if cause := swaggerCause(err); cause != "" {
			return fmt.Errorf("%w: caused by %s", err, swaggerCause(err))
		}
	}

	return err
}

func (podman *Podman) ImagePull(ctx context.Context, reference string) error {
	// nolint:bodyclose // already closed by Swagger-generated code
	_, _, err := podman.cli.ImagesApi.LibpodImagesPull(ctx, &swagger.ImagesApiLibpodImagesPullOpts{
		Reference: optional.NewString(reference),
	})

	// Enrich the error with it's cause if possible
	if err != nil {
		if cause := swaggerCause(err); cause != "" {
			return fmt.Errorf("%w: caused by %s", err, swaggerCause(err))
		}

		return err
	}

	// Due to how Swagger-generated API handles (and essentially ignores) pull errors, we need to check twice
	err = podman.ImageInspect(ctx, reference)

	// Enrich the error with it's cause if possible
	if err != nil {
		if cause := swaggerCause(err); cause != "" {
			return fmt.Errorf("%w: caused by %s", err, swaggerCause(err))
		}
	}

	return err
}

func (podman *Podman) ImageBuild(
	ctx context.Context,
	tarball io.Reader,
	input *ImageBuildInput,
) (<-chan string, <-chan error) {
	logChan := make(chan string)
	errChan := make(chan error)

	go func() {
		buildURL, err := url.Parse(podman.basePath + "/libpod/build")
		if err != nil {
			errChan <- err
			return
		}

		q := buildURL.Query()

		for _, tag := range input.Tags {
			q.Add("t", tag)
		}

		q.Add("dockerfile", input.Dockerfile)

		jsonArgs, err := json.Marshal(&input.BuildArgs)
		if err != nil {
			errChan <- err
			return
		}
		q.Add("buildargs", string(jsonArgs))

		q.Add("rm", "true")

		buildURL.RawQuery = q.Encode()

		req, err := http.NewRequestWithContext(ctx, "POST", buildURL.String(), tarball)
		if err != nil {
			errChan <- err
			return
		}
		req.Header.Set("Content-Type", "application/x-tar")

		resp, err := podman.httpClient.Do(req)
		if err != nil {
			errChan <- err
			return
		}

		if resp.StatusCode != http.StatusOK {
			errChan <- fmt.Errorf("%w: image build endpoint returned HTTP %d", ErrPodman, resp.StatusCode)
			return
		}

		unrollStream(resp.Body, logChan, errChan)

		if err := resp.Body.Close(); err != nil {
			errChan <- err
			return
		}

		errChan <- ErrDone
	}()

	return logChan, errChan
}

func (podman *Podman) ImageInspect(ctx context.Context, reference string) error {
	// nolint:bodyclose // already closed by Swagger-generated code
	_, resp, err := podman.cli.ImagesApi.LibpodInspectImage(ctx, reference)

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	// Enrich the error with it's cause if possible
	if err != nil {
		if cause := swaggerCause(err); cause != "" {
			return fmt.Errorf("%w: caused by %s", err, swaggerCause(err))
		}
	}

	return err
}

func (podman *Podman) ContainerCreate(
	ctx context.Context,
	input *ContainerCreateInput,
	name string,
) (*ContainerCreateOutput, error) {
	specGen := swagger.SpecGenerator{
		Name:       name,
		Entrypoint: input.Entrypoint,
		Command:    input.Command,
		Env:        input.Env,
		Image:      input.Image,
	}

	if strings.HasPrefix(input.Network, "container:") {
		specGen.Netns = &swagger.Namespace{
			Nsmode:  "container",
			String_: strings.TrimPrefix(input.Network, "container:"),
		}
	}

	for _, ourMount := range input.Mounts {
		var options []string

		if ourMount.ReadOnly {
			options = append(options, "ro")
		}

		switch ourMount.Type {
		case MountTypeBind:
			specGen.Mounts = append(specGen.Mounts, swagger.Mount{
				Type_:       "bind",
				Source:      ourMount.Source,
				Destination: ourMount.Target,
				Options:     options,
			})
		case MountTypeVolume:
			specGen.Volumes = append(specGen.Volumes, swagger.NamedVolume{
				Name:    ourMount.Source,
				Dest:    ourMount.Target,
				Options: options,
			})
		}
	}

	// nolint:bodyclose // already closed by Swagger-generated code
	cont, _, err := podman.cli.ContainersApi.LibpodCreateContainer(ctx, &swagger.ContainersApiLibpodCreateContainerOpts{
		Body: optional.NewInterface(&specGen),
	})
	if err != nil {
		cause := swaggerCause(err)
		if cause == "no such image" {
			return nil, fmt.Errorf("%w: no such image: \"%s\"", ErrPodman, input.Image)
		}

		return nil, fmt.Errorf("%w: caused by %s", err, cause)
	}

	return &ContainerCreateOutput{
		ID: cont.Id,
	}, nil
}

func (podman *Podman) ContainerStart(ctx context.Context, id string) error {
	// nolint:bodyclose // already closed by Swagger-generated code
	_, err := podman.cli.ContainersApi.LibpodStartContainer(ctx, id, &swagger.ContainersApiLibpodStartContainerOpts{})

	// Enrich the error with it's cause if possible
	if err != nil {
		if cause := swaggerCause(err); cause != "" {
			return fmt.Errorf("%w: caused by %s", err, swaggerCause(err))
		}
	}

	return err
}

func (podman *Podman) ContainerWait(ctx context.Context, id string) (<-chan ContainerWaitResult, <-chan error) {
	waitChan := make(chan ContainerWaitResult)
	errChan := make(chan error)

	go func() {
		// nolint:bodyclose // already closed by Swagger-generated code
		resp, _, err := podman.cli.ContainersApi.LibpodWaitContainer(ctx, id, &swagger.ContainersApiLibpodWaitContainerOpts{
			Condition: optional.NewString("stopped"),
		})

		if err != nil {
			// Enrich the error with it's cause if possible
			if cause := swaggerCause(err); cause != "" {
				err = fmt.Errorf("%w: caused by %s", err, swaggerCause(err))
			}

			errChan <- err
			return
		}

		result := ContainerWaitResult{
			StatusCode: resp.StatusCode,
		}

		if resp.Error_ != nil {
			result.Error = resp.Error_.Message
		}

		waitChan <- result
	}()

	return waitChan, errChan
}

func (podman *Podman) ContainerDelete(ctx context.Context, id string) error {
	// nolint:bodyclose // already closed by Swagger-generated code
	_, err := podman.cli.ContainersApi.LibpodRemoveContainer(ctx, id, &swagger.ContainersApiLibpodRemoveContainerOpts{
		Force: optional.NewBool(true),
		V:     optional.NewBool(true),
	})

	// Enrich the error with it's cause if possible
	if err != nil {
		if cause := swaggerCause(err); cause != "" {
			return fmt.Errorf("%w: caused by %s", err, swaggerCause(err))
		}
	}

	return err
}

func (podman *Podman) SystemInfo(ctx context.Context) (*SystemInfo, error) {
	// nolint:bodyclose // already closed by Swagger-generated code
	info, _, err := podman.cli.SystemApi.LibpodGetInfo(ctx)

	// Enrich the error with it's cause if possible
	if err != nil {
		if cause := swaggerCause(err); cause != "" {
			return nil, fmt.Errorf("%w: caused by %s", err, swaggerCause(err))
		}

		return nil, err
	}

	return &SystemInfo{
		TotalCPUs:        info.Host.Cpus,
		TotalMemoryBytes: info.Host.MemTotal,
	}, nil
}

func swaggerCause(err error) string {
	if swaggerError, ok := err.(swagger.GenericSwaggerError); ok {
		if parsedError, ok := swaggerError.Model().(swagger.InlineResponse400); ok {
			return parsedError.Cause
		}
	}

	return ""
}
