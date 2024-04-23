package executor

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/samber/lo"
	"io"
	"net/http"
)

type UploadDescriptor struct {
	url     string
	headers map[string]string
}

type HTTPSUploader struct {
	httpClient         *http.Client
	taskIdentification *api.TaskIdentification

	artifacts         *Artifacts
	uploadDescriptors map[string]*UploadDescriptor
	uploadedFiles     []*api.ArtifactFileInfo
}

func NewHTTPSUploader(
	ctx context.Context,
	taskIdentification *api.TaskIdentification,
	artifacts *Artifacts,
) (ArtifactUploader, error) {
	// Create a mapping between relative artifact paths and upload URLs
	uploadDescriptors := map[string]*UploadDescriptor{}

	// Generate URLs to which we'll upload the artifacts
	for _, uploadableFilesChunk := range lo.Chunk(artifacts.UploadableFiles(), 100) {
		request := &api.GenerateArtifactUploadURLsRequest{
			TaskIdentification: taskIdentification,
			Name:               artifacts.Name,
			Files:              uploadableFilesChunk,
		}

		response, err := client.CirrusClient.GenerateArtifactUploadURLs(ctx, request)
		if err != nil {
			return nil, err
		}

		if len(request.Files) != len(response.Urls) {
			return nil, fmt.Errorf("GenerateArtifactUploadURLs() RPC call returned invalid data:"+
				" requested %d URLs, got %d", len(request.Files), len(response.Urls))
		}

		for idx, url := range response.Urls {
			uploadDescriptors[request.Files[idx].Path] = &UploadDescriptor{
				url:     url.Url,
				headers: url.Headers,
			}
		}
	}

	return &HTTPSUploader{
		httpClient:         &http.Client{},
		taskIdentification: taskIdentification,
		artifacts:          artifacts,
		uploadDescriptors:  uploadDescriptors,
	}, nil
}

func (uploader *HTTPSUploader) Upload(ctx context.Context, artifact io.Reader, relativeArtifactPath string, size int64) error {
	uploadDescriptor, ok := uploader.uploadDescriptors[relativeArtifactPath]
	if !ok {
		return fmt.Errorf("no upload URL was generated for artifact path %s", relativeArtifactPath)
	}

	body := artifact
	if size == 0 {
		// According to the docs:
		// > The only way to explicitly say that the ContentLength is zero is to set the Body to nil.
		// Otherwise, the HTTP client will try to use chunked encoding which will lead to 501 (Not Implemented) for S3.
		body = nil
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadDescriptor.url, body)
	if err != nil {
		return err
	}

	httpRequest.Header.Set("Content-Type", "application/octet-stream")
	httpRequest.ContentLength = size
	for key, value := range uploadDescriptor.headers {
		httpRequest.Header.Set(key, value)
	}

	httpResponse, err := uploader.httpClient.Do(httpRequest)
	if err != nil {
		return err
	}

	if httpResponse.StatusCode != http.StatusOK && httpResponse.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to upload artifact file %s, HTTP status code: %d", relativeArtifactPath,
			httpResponse.StatusCode)
	}

	uploader.uploadedFiles = append(uploader.uploadedFiles, &api.ArtifactFileInfo{
		Path:        relativeArtifactPath,
		SizeInBytes: size,
	})

	return nil
}

func (uploader *HTTPSUploader) Finish(ctx context.Context) error {
	commitRequest := &api.CommitUploadedArtifactsRequest{
		TaskIdentification: uploader.taskIdentification,
		Name:               uploader.artifacts.Name,
		Type:               uploader.artifacts.Type,
		Format:             uploader.artifacts.Format,
		Files:              uploader.uploadedFiles,
	}
	_, err := client.CirrusClient.CommitUploadedArtifacts(ctx, commitRequest)
	if err != nil {
		return err
	}

	return nil
}
