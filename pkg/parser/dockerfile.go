package parser

import (
	"container/list"
	"context"
	"crypto/md5" //nolint:gosec // backwards compatibility
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/constants"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/dockerfile"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"
	"sort"
	"strings"
	"syscall"
)

var (
	ErrFailedToRetrieve = errors.New("failed to retrieve")
	ErrFailedToAnalyze  = errors.New("failed to analyze")
)

func (p *Parser) calculateDockerfileHashes(
	ctx context.Context,
	protoTasks []*api.Task,
	protoTaskToInstanceNode map[int64]*node.Node,
) error {
	for _, protoTask := range protoTasks {
		if protoTask.Instance == nil && p.missingInstancesAllowed {
			continue
		}

		dynamicInstance, err := anypb.UnmarshalNew(protoTask.Instance, proto.UnmarshalOptions{})

		if errors.Is(err, protoregistry.NotFound) {
			continue
		}

		if err != nil {
			return fmt.Errorf("%w: failed to unmarshal task's instance: %v", parsererror.ErrInternal, err)
		}

		reflectedInstance := dynamicInstance.ProtoReflect()

		// Pick up the "dockerfile:" field or skip the task
		dockerfileField := reflectedInstance.Descriptor().Fields().ByName("dockerfile")
		if dockerfileField == nil || !reflectedInstance.Has(dockerfileField) {
			continue
		}
		dockerfilePath := reflectedInstance.Get(dockerfileField).String()

		// Pick up the "docker_arguments:" field, if any
		dockerArguments := map[string]string{}

		dockerArgumentsField := reflectedInstance.Descriptor().Fields().ByName("docker_arguments")
		if dockerArgumentsField != nil && reflectedInstance.Has(dockerArgumentsField) {
			dockerArgumentsMap, ok := reflectedInstance.Get(dockerArgumentsField).Interface().(protoreflect.Map)
			if ok {
				dockerArgumentsMap.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
					dockerArguments[key.String()] = value.String()
					return true
				})
			}
		}

		// Retrieve a parser node associated with this instance
		// to generate line/column-specific errors and warnings
		instanceNode, ok := protoTaskToInstanceNode[protoTask.LocalGroupId]
		if !ok {
			continue
		}

		dockerfileNode := instanceNode.FindChild("dockerfile")
		if dockerfileNode == nil {
			continue
		}

		// Calculate the Dockerfile hash
		dockerfileHash, err := p.calculateDockerfileHash(ctx, dockerfilePath, dockerArguments, dockerfileNode,
			protoTask.Environment["CIRRUS_DOCKER_CONTEXT"], protoTask.Environment[constants.EnvironmentCirrusArch])
		if err != nil {
			return err
		}

		// Save the calculated hash in the properties for (1) the
		// service task creation routine and (2) the Cirrus Cloud
		protoTask.Metadata.Properties[metadataPropertyDockerfileHash] = dockerfileHash
	}

	return nil
}

func (p *Parser) calculateDockerfileHash(
	ctx context.Context,
	dockerfilePath string,
	dockerArguments map[string]string,
	dockerfileNode *node.Node,
	dockerContext string,
	cirrusArch string,
) (string, error) {
	dockerfileContents, err := p.fs.Get(ctx, dockerfilePath)
	if err != nil {
		return "", dockerfileNode.ParserError("%v %q: %v", ErrFailedToRetrieve, dockerfilePath, err)
	}

	//nolint:gosec // backwards compatibility
	oldHash := md5.New()
	newHash := sha256.New()

	// Calculate a shallow hash
	oldHash.Write(dockerfileContents)
	newHash.Write(dockerfileContents)

	hashableArgs := dockerArgumentsToString(dockerArguments)
	oldHash.Write([]byte(hashableArgs))
	newHash.Write([]byte(hashableArgs))

	if cirrusArch != "" && cirrusArch != "amd64" {
		oldHash.Write([]byte(cirrusArch))
		newHash.Write([]byte(cirrusArch))
	}

	// Try to calculate a deep hash
	sourcePaths, err := dockerfile.LocalContextSourcePaths(ctx, dockerfileContents, dockerArguments)
	if err != nil {
		p.parserKit.IssueRegistry.RegisterIssuef(api.Issue_WARNING, dockerfileNode.Line, dockerfileNode.Column,
			"%v %q: %v", ErrFailedToAnalyze, dockerfilePath, err)

		return hex.EncodeToString(oldHash.Sum([]byte{})), nil
	}

	var hashedAtLeastOneSource bool

	for _, sourcePath := range sourcePaths {
		if dockerContext != "" {
			sourcePath = p.fs.Join(dockerContext, sourcePath)
		}

		if err := find(ctx, p.fs, sourcePath, func(filePath string, fileContents []byte) {
			newHash.Write(fileContents)
			hashedAtLeastOneSource = true
		}); err != nil {
			p.parserKit.IssueRegistry.RegisterIssuef(api.Issue_WARNING, dockerfileNode.Line, dockerfileNode.Column, "%v", err)
		}
	}

	if hashedAtLeastOneSource {
		return hex.EncodeToString(newHash.Sum([]byte{})), nil
	}

	return hex.EncodeToString(oldHash.Sum([]byte{})), nil
}

func dockerArgumentsToString(buildArgs map[string]string) string {
	var flattenedArgs []string

	for key, value := range buildArgs {
		flattenedArgs = append(flattenedArgs, key+value)
	}

	sort.Strings(flattenedArgs)

	return strings.Join(flattenedArgs, ", ")
}

func find(ctx context.Context, fs fs.FileSystem, path string, cb func(path string, contents []byte)) error {
	todo := list.New()

	todo.PushBack(path)

	for todo.Len() > 0 {
		todoPath := todo.Remove(todo.Front()).(string)

		namesInDir, err := fs.ReadDir(ctx, todoPath)
		if err != nil {
			if errors.Is(err, syscall.ENOTDIR) {
				todoContents, err := fs.Get(ctx, todoPath)
				if err != nil {
					return fmt.Errorf("%w file %q referenced in ADD/COPY instruction: %v",
						ErrFailedToRetrieve, todoPath, err)
				}

				cb(todoPath, todoContents)

				continue
			}

			return fmt.Errorf("%w directory %q referenced in ADD/COPY instruction: %v",
				ErrFailedToRetrieve, todoPath, err)
		}

		for _, name := range namesInDir {
			todo.PushBack(fs.Join(todoPath, name))
		}
	}

	return nil
}
