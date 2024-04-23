package executor

import (
	"context"
	"fmt"
	"github.com/bmatcuk/doublestar"
	"github.com/cirruslabs/cirrus-cli/internal/agent/environment"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
)

type ArtifactUploader interface {
	Upload(ctx context.Context, artifact io.Reader, relativeArtifactPath string, size int64) error
	Finish(ctx context.Context) error
}

type InstantiateArtifactUploaderFunc func(
	ctx context.Context,
	taskIdentification *api.TaskIdentification,
	artifacts *Artifacts,
) (ArtifactUploader, error)

type Artifacts struct {
	Name     string
	Type     string
	Format   string
	patterns []*ProcessedPattern
}

type ProcessedPattern struct {
	Pattern string
	Paths   []*ProcessedPath
}

type ProcessedPath struct {
	absolutePath string
	relativePath string
	info         os.FileInfo
}

func NewArtifacts(
	name string,
	artifactsInstruction *api.ArtifactsInstruction,
	customEnv *environment.Environment,
) (*Artifacts, error) {
	workingDir := customEnv.Get("CIRRUS_WORKING_DIR")

	result := &Artifacts{
		Name:   name,
		Type:   artifactsInstruction.Type,
		Format: artifactsInstruction.Format,
	}

	for _, path := range artifactsInstruction.Paths {
		pattern := customEnv.ExpandText(path)
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(workingDir, pattern)
		}

		paths, err := doublestar.Glob(pattern)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to list artifacts")
		}

		processedPattern := &ProcessedPattern{
			Pattern: pattern,
		}

		// Ensure that the all resulting paths are scoped to the CIRRUS_WORKING_DIR
		for _, artifactPath := range paths {
			matcher := filepath.Join(workingDir, "**")
			matched, err := doublestar.PathMatch(matcher, artifactPath)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to match the path: %v", err)
			}
			if !matched {
				return nil, fmt.Errorf("%w: path %s should be relative to %s",
					ErrArtifactsPathOutsideWorkingDir, artifactPath, workingDir)
			}

			info, err := os.Stat(artifactPath)
			if err != nil {
				return nil, fmt.Errorf("failed to stat artifact at path %s", artifactPath)
			}

			relativeArtifactPath, err := filepath.Rel(workingDir, artifactPath)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get artifact relative path for %s", artifactPath)
			}
			relativeArtifactPath = filepath.ToSlash(relativeArtifactPath)

			processedPattern.Paths = append(processedPattern.Paths, &ProcessedPath{
				absolutePath: artifactPath,
				relativePath: relativeArtifactPath,
				info:         info,
			})
		}

		result.patterns = append(result.patterns, processedPattern)
	}

	return result, nil
}

func (artifacts *Artifacts) UploadableFiles() []*api.ArtifactFileInfo {
	var result []*api.ArtifactFileInfo

	for _, pattern := range artifacts.patterns {
		for _, path := range pattern.Paths {
			if path.info.IsDir() {
				continue
			}

			result = append(result, &api.ArtifactFileInfo{
				Path:        path.relativePath,
				SizeInBytes: path.info.Size(),
			})
		}
	}

	return result
}
