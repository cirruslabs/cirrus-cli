package rpc

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/pathsafe"
	"io"
	"os"
	"path/filepath"
)

func (r *RPC) UploadArtifacts(stream api.CirrusCIService_UploadArtifactsServer) error {
	var currentArtifactName string
	var lastSeenArtifactPath string

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

			artifactPath := filepath.Join(r.artifactsDir, currentArtifactName, obj.Chunk.ArtifactPath)

			var maybeTruncate int

			if obj.Chunk.ArtifactPath != lastSeenArtifactPath {
				maybeTruncate = os.O_TRUNC

				lastSeenArtifactPath = obj.Chunk.ArtifactPath
			}

			if err := os.MkdirAll(filepath.Dir(artifactPath), 0700); err != nil {
				return err
			}

			artifact, err := os.OpenFile(artifactPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE|maybeTruncate, 0600)
			if err != nil {
				return err
			}

			if _, err := artifact.Write(obj.Chunk.Data); err != nil {
				return err
			}

			if err := artifact.Close(); err != nil {
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
