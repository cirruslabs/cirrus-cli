package executor

import (
	"bufio"
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/pkg/errors"
	"io"
)

const readBufferSizeBytes = 1024 * 1024

type GRPCUploader struct {
	taskIdentification *api.TaskIdentification

	client     api.CirrusCIService_UploadArtifactsClient
	readBuffer []byte
}

func NewGRPCUploader(
	ctx context.Context,
	taskIdentification *api.TaskIdentification,
	artifacts *Artifacts,
) (ArtifactUploader, error) {
	client, err := client.CirrusClient.UploadArtifacts(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize artifacts upload client")
	}

	err = client.Send(&api.ArtifactEntry{
		Value: &api.ArtifactEntry_ArtifactsUpload_{
			ArtifactsUpload: &api.ArtifactEntry_ArtifactsUpload{
				TaskIdentification: taskIdentification,
				Name:               artifacts.Name,
				Type:               artifacts.Type,
				Format:             artifacts.Format,
			},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize artifacts upload")
	}

	return &GRPCUploader{
		taskIdentification: taskIdentification,

		client:     client,
		readBuffer: make([]byte, readBufferSizeBytes),
	}, nil
}

func (uploader *GRPCUploader) Upload(ctx context.Context, artifact io.Reader, relativeArtifactPath string, _ int64) error {
	bufferedArtifactReader := bufio.NewReaderSize(artifact, readBufferSizeBytes)

	for {
		n, err := bufferedArtifactReader.Read(uploader.readBuffer)
		if n == 0 || err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrapf(err, "failed to read artifact file %s", relativeArtifactPath)
		}

		err = uploader.client.Send(&api.ArtifactEntry{
			Value: &api.ArtifactEntry_Chunk{
				Chunk: &api.ArtifactEntry_ArtifactChunk{
					ArtifactPath: relativeArtifactPath,
					Data:         uploader.readBuffer[:n],
				},
			},
		})
		if err != nil {
			return errors.Wrapf(err, "failed to upload artifact file %s", relativeArtifactPath)
		}
	}

	return nil
}

func (uploader *GRPCUploader) Finish(ctx context.Context) error {
	_, err := uploader.client.CloseAndRecv()
	if err != nil {
		return errors.Wrap(err, "failed to finalize upload stream")
	}

	return nil
}
