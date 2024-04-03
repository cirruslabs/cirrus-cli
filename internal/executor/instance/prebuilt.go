package instance

import (
	"archive/tar"
	"context"
	"errors"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"go.opentelemetry.io/otel/attribute"
	"io"
	"os"
	"path/filepath"
)

type PrebuiltInstance struct {
	Image      string
	Dockerfile string
	Arguments  map[string]string
}

func CreateTempArchive(dir string) (string, error) {
	tmpFile, err := os.CreateTemp("", "cirrus-prebuilt-archive-")
	if err != nil {
		return "", err
	}

	archive := tar.NewWriter(tmpFile)

	if err := filepath.Walk(dir, func(path string, fileInfo os.FileInfo, err error) error {
		// Handle possible error that occurred when reading this directory entry information
		if err != nil {
			return err
		}

		// We clearly don't want any directories here (because tar)
		// and probably not interested in special files for now
		if !fileInfo.Mode().IsRegular() {
			return nil
		}

		header, err := tar.FileInfoHeader(fileInfo, fileInfo.Name())
		if err != nil {
			return err
		}

		// Since os.FileInfo doesn't contain the full path to a file
		// we need to manually update the Name field in the header
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write file header
		if err := archive.WriteHeader(header); err != nil {
			return err
		}

		// Write file contents
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := io.Copy(archive, file); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return "", err
	}

	if err := archive.Close(); err != nil {
		return "", err
	}

	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

func (prebuilt *PrebuiltInstance) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("image", prebuilt.Image),
		attribute.String("instance_type", "prebuilt"),
	}
}

func (prebuilt *PrebuiltInstance) Run(ctx context.Context, config *runconfig.RunConfig) error {
	logger := config.Logger()
	backend, err := config.GetContainerBackend()
	if err != nil {
		return err
	}

	// Check if the image we're about to build is available locally
	if err = backend.ImageInspect(ctx, prebuilt.Image); err == nil {
		logger.Infof("Re-using local image %s...", prebuilt.Image)
		return nil
	}

	// The image is not available locally, try to pull it
	logger.Infof("Pulling image %s...", prebuilt.Image)
	if err := backend.ImagePull(ctx, prebuilt.Image, nil); err == nil {
		logger.Infof("Using pulled image %s...", prebuilt.Image)
		return nil
	}

	logger.Infof("Image %s is not available locally nor remotely, building it...", prebuilt.Image)

	// Create an archive with the build context
	archivePath, err := CreateTempArchive(config.ProjectDir)
	if err != nil {
		return err
	}
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer func() {
		// Don't bother with catching the error since the file may be already closed by a container backend
		_ = file.Close()

		if err := os.Remove(archivePath); err != nil {
			logger.Warnf("while removing temporary archive file: %v", err)
		}
	}()

	// Build the image
	logChan, errChan := backend.ImageBuild(ctx, file, &containerbackend.ImageBuildInput{
		Tags:       []string{prebuilt.Image},
		Dockerfile: prebuilt.Dockerfile,
		BuildArgs:  prebuilt.Arguments,
		Pull:       !config.ContainerOptions.LazyPull,
	})

Outer:
	for {
		select {
		case line := <-logChan:
			logger.Infof("%s", line)
		case err := <-errChan:
			if errors.Is(containerbackend.ErrDone, err) {
				break Outer
			}

			return err
		}
	}

	// Push the image (if needed)
	if config.ContainerOptions.DockerfileImagePush {
		return backend.ImagePush(ctx, prebuilt.Image)
	}

	return nil
}

func (prebuilt *PrebuiltInstance) WorkingDirectory(projectDir string, dirtyMode bool) string {
	return ""
}

func (prebuilt *PrebuiltInstance) Close(context.Context) error {
	return nil
}
