package executor

import (
	"context"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/cirruslabs/cirrus-ci-annotations"
	"github.com/cirruslabs/cirrus-ci-annotations/model"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/internal/agent/environment"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
)

var ErrArtifactsPathOutsideWorkingDir = errors.New("path is outside of CIRRUS_WORKING_DIR")

func (executor *Executor) UploadArtifacts(
	ctx context.Context,
	logUploader *LogUploader,
	name string,
	artifactsInstruction *api.ArtifactsInstruction,
	customEnv *environment.Environment,
) bool {
	// Check if we need to upload anything at all
	if len(artifactsInstruction.Paths) == 0 {
		fmt.Fprintln(logUploader, "Skipping artifacts upload because there are no paths specified...")

		return true
	}

	artifacts, err := NewArtifacts(name, artifactsInstruction, customEnv)
	if err != nil {
		fmt.Fprintf(logUploader, "Failed to upload artifacts: %v", err)

		return false
	}

	// Upload artifacts: try first via HTTPS, then fallback via gRPC if not implemented
	err = executor.uploadArtifactsWithRetries(ctx, NewHTTPSUploader, logUploader, artifacts)
	if errStatus, ok := status.FromError(err); ok {
		if errStatus.Code() == codes.Unimplemented {
			fmt.Fprintf(logUploader, "Artifact upload via pre-signed URLs is not supported! Falling back to gRPC...\n")
			err = executor.uploadArtifactsWithRetries(ctx, NewGRPCUploader, logUploader, artifacts)
		}
	}

	if err != nil {
		fmt.Fprintf(logUploader, "Failed to upload artifacts: %s\n", err)
		return false
	}

	// Process and upload annotations
	if artifactsInstruction.Format != "" {
		return executor.processAndUploadAnnotations(ctx, customEnv.Get("CIRRUS_WORKING_DIR"),
			artifacts.UploadableFiles(), logUploader, artifactsInstruction.Format)
	}

	return true
}

func (executor *Executor) uploadArtifactsWithRetries(ctx context.Context, instantiateArtifactUploader InstantiateArtifactUploaderFunc, logUploader *LogUploader, artifacts *Artifacts) (err error) {
	err = retry.Do(
		func() error {
			artifactUploader, err := instantiateArtifactUploader(ctx, executor.taskIdentification(), artifacts)
			if err != nil {
				return err
			}

			if err := uploadArtifacts(ctx, artifacts, logUploader, artifactUploader); err != nil {
				return err
			}

			if err := artifactUploader.Finish(ctx); err != nil {
				fmt.Fprintf(logUploader, "Failed to finalize artifact uploader: %v\n", err)

				return err
			}

			return nil
		}, retry.OnRetry(func(n uint, err error) {
			fmt.Fprintf(logUploader, "Failed to upload artifacts: %v\n", err)
			fmt.Fprintln(logUploader, "Re-trying to artifacts upload...")
		}),
		retry.Attempts(2),
		retry.Context(ctx),
		retry.RetryIf(func(err error) bool {
			if errors.Is(err, ErrArtifactsPathOutsideWorkingDir) {
				return false
			}

			if status, ok := status.FromError(err); ok {
				if status.Code() == codes.Unimplemented {
					return false
				}
			}

			return true
		}),
		retry.LastErrorOnly(true),
	)
	if err != nil {
		fmt.Fprintf(logUploader, "Failed to upload artifacts after multiple tries: %v\n", err)

		return err
	}

	return nil
}

func uploadArtifacts(
	ctx context.Context,
	artifacts *Artifacts,
	logUploader *LogUploader,
	artifactUploader ArtifactUploader,
) error {
	for _, pattern := range artifacts.patterns {
		fmt.Fprintf(logUploader, "Uploading %d artifacts for %s\n", len(pattern.Paths), pattern.Pattern)

		for _, artifactPath := range pattern.Paths {
			if artifactPath.info.IsDir() {
				fmt.Fprintf(logUploader, "Skipping uploading of '%s' because it's a folder\n",
					artifactPath.absolutePath)
				continue
			}

			if artifactPath.info.Size() > 100*humanize.MByte {
				fmt.Fprintf(logUploader, "Uploading a quite hefty artifact '%s' of size %s\n",
					artifactPath.absolutePath, humanize.Bytes(uint64(artifactPath.info.Size())))
			}

			artifactFile, err := os.Open(artifactPath.absolutePath)
			if err != nil {
				return errors.Wrapf(err, "failed to read artifact file %s", artifactPath.absolutePath)
			}

			err = artifactUploader.Upload(ctx, artifactFile, artifactPath.relativePath, artifactPath.info.Size())
			if err != nil {
				_ = artifactFile.Close()
				return err
			}

			_ = artifactFile.Close()

			fmt.Fprintf(logUploader, "Uploaded %s\n", artifactPath.absolutePath)
		}
	}

	return nil
}

func (executor *Executor) processAndUploadAnnotations(
	ctx context.Context,
	workingDir string,
	uploadedArtifacts []*api.ArtifactFileInfo,
	logUploader *LogUploader,
	format string,
) bool {
	var allAnnotations []model.Annotation

	for _, uploadedArtifact := range uploadedArtifacts {
		fmt.Fprintf(logUploader, "Trying to parse annotations for %s format\n", format)

		err, artifactAnnotations := annotations.ParseAnnotations(format, uploadedArtifact.Path)
		if err != nil {
			fmt.Fprintf(logUploader, "failed to create annotations from %s: %v", uploadedArtifact.Path, err)

			return false
		}

		allAnnotations = append(allAnnotations, artifactAnnotations...)
	}

	if len(allAnnotations) == 0 {
		return true
	}

	normalizedAnnotations, err := annotations.NormalizeAnnotations(workingDir, allAnnotations)
	if err != nil {
		fmt.Fprintf(logUploader, "Failed to validate annotations: %v\n", err)
	}
	protoAnnotations := ConvertAnnotations(normalizedAnnotations)
	reportAnnotationsCommandRequest := api.ReportAnnotationsCommandRequest{
		TaskIdentification: executor.taskIdentification(),
		Annotations:        protoAnnotations,
	}

	err = retry.Do(
		func() error {
			_, err = client.CirrusClient.ReportAnnotations(ctx, &reportAnnotationsCommandRequest)
			return err
		}, retry.OnRetry(func(n uint, err error) {
			fmt.Fprintf(logUploader, "Failed to report %d annotations: %s\n", len(normalizedAnnotations), err)
			fmt.Fprintln(logUploader, "Retrying...")
		}),
		retry.Attempts(2),
		retry.Context(ctx),
	)
	if err != nil {
		fmt.Fprintf(logUploader, "Still failed to report %d annotations: %s. Ignoring...\n",
			len(normalizedAnnotations), err)

		return true
	}

	fmt.Fprintf(logUploader, "Reported %d annotations!\n", len(normalizedAnnotations))

	return true
}
