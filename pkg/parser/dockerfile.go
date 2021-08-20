package parser

import (
	"container/list"
	"context"
	"crypto/md5" // nolint:gosec // backwards compatibility
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/dockerfile"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"
	"sort"
	"strings"
	"syscall"
)

func (p *Parser) calculateDockerfileHashes(ctx context.Context, protoTasks []*api.Task) error {
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

		// Calculate the Dockerfile hash
		dockerfileHash, err := p.calculateDockerfileHash(ctx, dockerfilePath, dockerArguments)
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
) (string, error) {
	dockerfileContents, err := p.fs.Get(ctx, dockerfilePath)
	if err != nil {
		return "", parsererror.NewRich(1, 1, "failed to retrieve %q: %v",
			dockerfilePath, err)
	}

	// nolint:gosec // backwards compatibility
	oldHash := md5.New()
	newHash := sha256.New()

	// Calculate a shallow hash
	oldHash.Write(dockerfileContents)
	newHash.Write(dockerfileContents)

	hashableArgs := dockerArgumentsToString(dockerArguments)
	oldHash.Write([]byte(hashableArgs))
	newHash.Write([]byte(hashableArgs))

	// Try to calculate a deep hash
	sourcePaths, err := dockerfile.LocalContextSourcePaths(ctx, dockerfileContents, dockerArguments)
	if err != nil {
		p.registerIssuef(api.Issue_WARNING, 1, 1, "failed to analyze %q: %v", dockerfilePath, err)

		return hex.EncodeToString(oldHash.Sum([]byte{})), nil
	}

	var hashedAtLeastOneSource bool

	for _, sourcePath := range sourcePaths {
		if err := find(ctx, p.fs, sourcePath, func(filePath string, fileContents []byte) {
			newHash.Write(fileContents)
			hashedAtLeastOneSource = true
		}); err != nil {
			return "", err
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

	for todoEntry := todo.Front(); todoEntry != nil; todoEntry = todoEntry.Next() {
		todoPath := todo.Remove(todoEntry).(string)

		namesInDir, err := fs.ReadDir(ctx, todoPath)
		if err != nil {
			if errors.Is(err, syscall.ENOTDIR) {
				todoContents, err := fs.Get(ctx, todoPath)
				if err != nil {
					return parsererror.NewRich(1, 1, "failed to retrieve %q: %v", todoPath, err)
				}

				cb(todoPath, todoContents)

				continue
			}

			return parsererror.NewRich(1, 1, "failed to retrieve %q: %v", todoPath, err)
		}

		for _, name := range namesInDir {
			todo.PushBack(fs.Join(todoPath, name))
		}
	}

	return nil
}
