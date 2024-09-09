package containerbackend

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/antihax/optional"
	"github.com/avast/retry-go/v4"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend/podman"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/podmanapi/pkg/swagger"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/google/uuid"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var ErrPodman = errors.New("Podman error")

type Podman struct {
	cmd        *exec.Cmd
	basePath   string
	httpClient *http.Client
	cli        *swagger.APIClient

	version *semver.Version
}

func NewPodman() (ContainerBackend, error) {
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("podman-%s.sock", uuid.New().String()))
	socketURI := fmt.Sprintf("unix://%s", socketPath)

	cmd := exec.Command("podman", "system", "service", "-t", "0", socketURI)

	// Prevent the signals sent to the CLI from reaching the Podman process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	err := retry.Do(func() error {
		_, err := os.Stat(socketPath)
		return err
	})
	if err != nil {
		return nil, err
	}

	podman := &Podman{
		cmd:      cmd,
		basePath: "http://d/v1.0.0",
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
				},
			},
		},
	}

	// Create Podman client
	podman.cli = swagger.NewAPIClient(&swagger.Configuration{
		BasePath:   podman.basePath,
		HTTPClient: podman.httpClient,
	})

	// Query server's version and activate bug workarounds (if applicable)
	version, err := podman.SystemInfo(context.Background())
	if err != nil {
		return nil, err
	}

	podman.version, _ = semver.NewVersion(version.Version)

	return podman, nil
}

func (backend *Podman) Close() error {
	doneChan := make(chan error)

	go func() {
		doneChan <- backend.cmd.Wait()
	}()

	var interruptSent, killSent bool

	for {
		select {
		case <-time.After(time.Second):
			if !killSent {
				if err := backend.cmd.Process.Kill(); err != nil {
					return err
				}
				killSent = true
			}
		case err := <-doneChan:
			// Podman < 4.1.0 and lower doesn't seem to
			// exit cleanly, yielding exit code 1.
			//
			// So, let's ignore that since it's a pretty
			// old Podman version anyway.
			//
			// From the changelog:
			//
			// >Podman now exits cleanly (with exit code 0) after receiving SIGTERM.[1]
			//
			// [1]: https://github.com/containers/podman/releases/tag/v4.1.0
			if backend.version != nil {
				execError := &exec.ExitError{}
				if errors.As(err, &execError) && execError.ExitCode() == 1 &&
					backend.version.LessThan(semver.MustParse("v4.1.0")) {
					return nil
				}
			}

			return err
		default:
			if !interruptSent {
				if err := backend.cmd.Process.Signal(syscall.SIGTERM); err != nil {
					return err
				}
				interruptSent = true
			}
		}
	}
}

func (backend *Podman) VolumeCreate(ctx context.Context, name string) error {
	//nolint:bodyclose // already closed by Swagger-generated code
	_, _, err := backend.cli.VolumesApi.LibpodCreateVolume(ctx, &swagger.VolumesApiLibpodCreateVolumeOpts{
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

func (backend *Podman) VolumeInspect(ctx context.Context, name string) error {
	//nolint:bodyclose // already closed by Swagger-generated code
	_, resp, err := backend.cli.VolumesApi.LibpodInspectVolume(ctx, name)

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

func (backend *Podman) VolumeDelete(ctx context.Context, name string) error {
	//nolint:bodyclose // already closed by Swagger-generated code
	_, err := backend.cli.VolumesApi.LibpodRemoveVolume(ctx, name, &swagger.VolumesApiLibpodRemoveVolumeOpts{
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

func (backend *Podman) ImagePull(ctx context.Context, reference string, _ *api.Architecture) error {
	//nolint:bodyclose // already closed by Swagger-generated code
	_, _, err := backend.cli.ImagesApi.LibpodImagesPull(ctx, &swagger.ImagesApiLibpodImagesPullOpts{
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
	err = backend.ImageInspect(ctx, reference)

	// Enrich the error with it's cause if possible
	if err != nil {
		if cause := swaggerCause(err); cause != "" {
			return fmt.Errorf("%w: caused by %s", err, swaggerCause(err))
		}
	}

	return err
}

func (backend *Podman) ImagePush(ctx context.Context, reference string) error {
	auth, err := podman.XRegistryAuthForImage(reference)
	if err != nil {
		return err
	}

	//nolint:bodyclose // already closed by Swagger-generated code
	_, _, err = backend.cli.ImagesApi.LibpodPushImage(ctx, reference, &swagger.ImagesApiLibpodPushImageOpts{
		Destination:   optional.NewString(reference),
		XRegistryAuth: optional.NewString(auth),
	})

	// Enrich the error with it's cause if possible
	if err != nil {
		if cause := swaggerCause(err); cause != "" {
			return fmt.Errorf("%w: caused by %s", err, swaggerCause(err))
		}

		return err
	}

	return nil
}

func (backend *Podman) ImageBuild(
	ctx context.Context,
	tarball io.Reader,
	input *ImageBuildInput,
) (<-chan string, <-chan error) {
	logChan := make(chan string)
	errChan := make(chan error)

	go func() {
		buildURL, err := url.Parse(backend.basePath + "/libpod/build")
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

		if input.Pull {
			q.Add("pull", "true")
		}

		q.Add("rm", "true")

		buildURL.RawQuery = q.Encode()

		req, err := http.NewRequestWithContext(ctx, "POST", buildURL.String(), tarball)
		if err != nil {
			errChan <- err
			return
		}
		req.Header.Set("Content-Type", "application/x-tar")

		resp, err := backend.httpClient.Do(req)
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

		// Work around https://github.com/containers/buildah/issues/1034
		if len(input.Tags) > 0 {
			tagParts := strings.Split(input.Tags[0], ":")
			const expectedNumberOfTagParts = 2

			if len(tagParts) == expectedNumberOfTagParts {
				//nolint:bodyclose // already closed by Swagger-generated code
				_, _ = backend.cli.ImagesApi.LibpodTagImage(ctx, "localhost/"+input.Tags[0], &swagger.ImagesApiLibpodTagImageOpts{
					Repo: optional.NewString(tagParts[0]),
					Tag:  optional.NewString(tagParts[1]),
				})
			}
		}

		errChan <- ErrDone
	}()

	return logChan, errChan
}

func (backend *Podman) ImageInspect(ctx context.Context, reference string) error {
	//nolint:bodyclose // already closed by Swagger-generated code
	_, resp, err := backend.cli.ImagesApi.LibpodInspectImage(ctx, reference)

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

func (backend *Podman) ImageDelete(ctx context.Context, reference string) error {
	//nolint:bodyclose // already closed by Swagger-generated code
	_, resp, err := backend.cli.ImagesApi.LibpodRemoveImage(ctx, reference, &swagger.ImagesApiLibpodRemoveImageOpts{})

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

func (backend *Podman) ContainerCreate(
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
		Privileged: input.Privileged,
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

	if input.DisableSELinux {
		specGen.SelinuxOpts = []string{"disable"}
	}

	//nolint:bodyclose // already closed by Swagger-generated code
	cont, _, err := backend.cli.ContainersApi.LibpodCreateContainer(ctx, &swagger.ContainersApiLibpodCreateContainerOpts{
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

func (backend *Podman) ContainerStart(ctx context.Context, id string) error {
	//nolint:bodyclose // already closed by Swagger-generated code
	_, err := backend.cli.ContainersApi.LibpodStartContainer(ctx, id, &swagger.ContainersApiLibpodStartContainerOpts{})

	// Enrich the error with it's cause if possible
	if err != nil {
		if cause := swaggerCause(err); cause != "" {
			return fmt.Errorf("%w: caused by %s", err, swaggerCause(err))
		}
	}

	return err
}

func (backend *Podman) ContainerWait(ctx context.Context, id string) (<-chan ContainerWaitResult, <-chan error) {
	waitChan := make(chan ContainerWaitResult)
	errChan := make(chan error)

	go func() {
		condition := "stopped"

		if backend.version != nil && backend.version.Equal(semver.MustParse("v3.0.0")) {
			// https://github.com/containers/podman/blob/v3.0.0/libpod/define/containerstate.go#L22
			condition = "4"
		}

		//nolint:bodyclose // already closed by Swagger-generated code
		resp, _, err := backend.cli.ContainersApi.LibpodWaitContainer(ctx, id, &swagger.ContainersApiLibpodWaitContainerOpts{
			Condition: optional.NewString(condition),
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

func (backend *Podman) ContainerLogs(ctx context.Context, id string) (<-chan string, error) {
	logChan := make(chan string, containerLogsChannelSize)

	buildURL, err := url.Parse(backend.basePath + "/containers/" + id + "/logs")
	if err != nil {
		return nil, err
	}

	q := buildURL.Query()
	q.Add("stdout", "true")
	q.Add("stderr", "true")
	q.Add("follow", "true")
	buildURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", buildURL.String(), nil)
	if err != nil {
		return nil, err
	}

	//nolint:bodyclose // it will be closed in the first Goroutine below
	resp, err := backend.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: container logs endpoint returned HTTP %d", ErrPodman, resp.StatusCode)
	}

	pipeReader, pipeWriter := io.Pipe()

	go func() {
		_, _ = stdcopy.StdCopy(pipeWriter, pipeWriter, resp.Body)
		_ = pipeWriter.Close()
		_ = resp.Body.Close()
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

func (backend *Podman) ContainerDelete(ctx context.Context, id string) error {
	//nolint:bodyclose // already closed by Swagger-generated code
	_, err := backend.cli.ContainersApi.LibpodRemoveContainer(ctx, id, &swagger.ContainersApiLibpodRemoveContainerOpts{
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

func (backend *Podman) SystemInfo(ctx context.Context) (*SystemInfo, error) {
	//nolint:bodyclose // already closed by Swagger-generated code
	info, _, err := backend.cli.SystemApi.LibpodGetInfo(ctx)

	// Enrich the error with it's cause if possible
	if err != nil {
		if cause := swaggerCause(err); cause != "" {
			return nil, fmt.Errorf("%w: caused by %s", err, swaggerCause(err))
		}

		return nil, err
	}

	var version string

	if info.Version != nil {
		version = info.Version.Version
	}

	return &SystemInfo{
		Version:          version,
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
