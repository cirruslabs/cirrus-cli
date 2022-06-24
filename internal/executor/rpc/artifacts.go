package rpc

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/pathsafe"
	"io"
	"os"
	"path/filepath"
)

func (r *RPC) UploadArtifacts(stream api.CirrusCIService_UploadArtifactsServer) (err error) {
	var currentArtifactName string
	artifactWriter := NewArtifactWriter(r.artifactsDir)
	defer func() {
		if awErr := artifactWriter.Close(); awErr != nil && err == nil {
			err = awErr
		}
	}()

	for {
		artifactEntry, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			r.logger.Warnf("error while receiving artifacts: %v", err)
			return err
		}

		if r.artifactsDir == "" {
			continue
		}

		switch obj := artifactEntry.Value.(type) {
		case *api.ArtifactEntry_ArtifactsUpload_:
			currentArtifactName = obj.ArtifactsUpload.Name
		case *api.ArtifactEntry_Chunk:
			if !pathsafe.IsPathSafe(currentArtifactName) {
				continue
			}

			artifactPath := filepath.Join(currentArtifactName, obj.Chunk.ArtifactPath)

			if _, err := artifactWriter.WriteTo(artifactPath, obj.Chunk.Data); err != nil {
				return err
			}
		}
	}

	if err := stream.SendAndClose(&api.UploadArtifactsResponse{}); err != nil {
		r.logger.Warnf("error while closing artifacts stream: %v", err)
		return err
	}

	return nil
}

type ArtifactWriter struct {
	baseDir     string
	currentPath string
	currentFile *os.File
}

func NewArtifactWriter(baseDir string) *ArtifactWriter {
	return &ArtifactWriter{
		baseDir:     baseDir,
		currentPath: "",
		currentFile: nil,
	}
}

func (aw *ArtifactWriter) WriteTo(name string, b []byte) (int, error) {
	path := filepath.Join(aw.baseDir, name)

	// nolint:nestif // doesn't look that complicated
	if aw.currentFile == nil || aw.currentPath != path {
		if aw.currentFile != nil {
			if err := aw.currentFile.Close(); err != nil {
				return 0, err
			}
		}

		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return 0, err
		}

		var err error

		aw.currentFile, err = os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return 0, err
		}

		aw.currentPath = path
	}

	return aw.currentFile.Write(b)
}

func (aw *ArtifactWriter) Close() error {
	if aw.currentFile == nil {
		return nil
	}

	return aw.currentFile.Close()
}
