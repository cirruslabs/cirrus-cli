package instance

import (
	"archive/tar"
	"bufio"
	"context"
	"encoding/json"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type PrebuiltInstance struct {
	Image      string
	Dockerfile string
	Arguments  map[string]string
}

func CreateTempArchive(dir string) (string, error) {
	tmpFile, err := ioutil.TempFile("", "cirrus-prebuilt-archive-")
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

func (prebuilt *PrebuiltInstance) Run(ctx context.Context, config *RunConfig) error {
	logger := config.Logger

	// Create a Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

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
		if err := file.Close(); err != nil {
			logger.Warnf("while closing temporary archive file: %v", err)
		}

		if err := os.Remove(archivePath); err != nil {
			logger.Warnf("while removing temporary archive file: %v", err)
		}
	}()

	// Deal with ImageBuildOptions's BuildArgs field quirks
	// since we don't differentiate between empty and missing
	// option values
	pointyArguments := make(map[string]*string)
	for key, value := range prebuilt.Arguments {
		valueCopy := value
		pointyArguments[key] = &valueCopy
	}

	// Build the image
	buildProgress, err := cli.ImageBuild(ctx, file, types.ImageBuildOptions{
		Tags:       []string{prebuilt.Image},
		Dockerfile: prebuilt.Dockerfile,
		BuildArgs:  pointyArguments,
		Remove:     true,
	})
	if err != nil {
		return err
	}

	buildProgressReader := bufio.NewReader(buildProgress.Body)

	for {
		// Docker build progress is line-based
		line, _, err := buildProgressReader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// Each line is a JSON object with the actual message wrapped in it
		msg := &struct {
			Stream string
		}{}
		if err := json.Unmarshal(line, &msg); err != nil {
			return err
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

		logger.Debugf("%s", progressMessage)
	}

	if err := buildProgress.Body.Close(); err != nil {
		return err
	}

	return nil
}

func (prebuilt *PrebuiltInstance) WorkingDirectory(projectDir string, dirtyMode bool) string {
	return ""
}
