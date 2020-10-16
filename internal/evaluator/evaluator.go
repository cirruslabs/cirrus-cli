package evaluator

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/larker"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/dummy"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/github"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"net"
	"strings"
)

type ConfigurationEvaluatorServiceServer struct {
	// must be embedded to have forward compatible implementations
	api.UnimplementedCirrusConfigurationEvaluatorServiceServer
}

func Serve(ctx context.Context, lis net.Listener) error {
	server := grpc.NewServer()

	api.RegisterCirrusConfigurationEvaluatorServiceServer(server, &ConfigurationEvaluatorServiceServer{})

	errChan := make(chan error)

	go func() {
		errChan <- server.Serve(lis)
	}()

	select {
	case <-ctx.Done():
		server.GracefulStop()

		if err := <-errChan; err != nil {
			return err
		}

		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

func fsFromEnvironment(env map[string]string) (fs fs.FileSystem) {
	// Fallback to dummy filesystem implementation
	fs = dummy.New()

	// Use GitHub filesystem if all the required variables are present
	owner, ok := env["CIRRUS_REPO_OWNER"]
	if !ok {
		return
	}
	repo, ok := env["CIRRUS_REPO_NAME"]
	if !ok {
		return
	}
	reference, ok := env["CIRRUS_CHANGE_IN_REPO"]
	if !ok {
		return
	}
	token, ok := env["CIRRUS_REPO_CLONE_TOKEN"]
	if !ok {
		return
	}

	return github.New(owner, repo, reference, token)
}

func (r *ConfigurationEvaluatorServiceServer) EvaluateConfig(
	ctx context.Context,
	request *api.EvaluateConfigRequest,
) (*api.EvaluateConfigResponse, error) {
	var yamlConfigs []string

	// Register YAML configuration (if any)
	if request.YamlConfig != "" {
		yamlConfigs = append(yamlConfigs, request.YamlConfig)
	}

	fs := fsFromEnvironment(request.Environment)

	// Run Starlark script and register generated YAML configuration (if any)
	if request.StarlarkConfig != "" {
		lrk := larker.New(
			larker.WithFileSystem(fs),
			larker.WithEnvironment(request.Environment),
		)

		generatedYamlConfig, err := lrk.Main(ctx, request.StarlarkConfig)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		yamlConfigs = append(yamlConfigs, generatedYamlConfig)
	}

	additionalInstances := make(map[string]protoreflect.MessageDescriptor)

	descriptorSet := request.AdditionalInstancesInfo.GetDescriptor_()
	if descriptorSet != nil {
		// convert protobuf descriptors to proto reflections
		// in order to pass additional instances provided dynamically by the request
		files, err := protodesc.NewFiles(descriptorSet)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		for fieldName, fqn := range request.AdditionalInstancesInfo.Instances {
			descriptor, err := files.FindDescriptorByName(protoreflect.FullName(fqn))
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to find %s type: %v", fqn, err))
			}
			if md, _ := descriptor.(protoreflect.MessageDescriptor); md != nil {
				additionalInstances[fieldName] = md
			}
		}
	}

	// Parse combined YAML
	p := parser.New(
		parser.WithEnvironment(request.Environment),
		parser.WithFileSystem(fs),
		parser.WithAdditionalInstances(additionalInstances),
	)

	result, err := p.Parse(ctx, strings.Join(yamlConfigs, "\n"))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if len(result.Errors) != 0 {
		return nil, status.Error(codes.InvalidArgument, result.Errors[0])
	}

	return &api.EvaluateConfigResponse{Tasks: result.Tasks}, nil
}
