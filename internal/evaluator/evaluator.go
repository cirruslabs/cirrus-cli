package evaluator

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/larker"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/dummy"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/github"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
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

	additionalInstances, err := TransformAdditionalInstances(request.AdditionalInstancesInfo)
	if err != nil {
		return nil, err
	}

	// Parse combined YAML
	p := parser.New(
		parser.WithEnvironment(request.Environment),
		parser.WithAffectedFiles(request.AffectedFiles),
		parser.WithFileSystem(fs),
		parser.WithAdditionalInstances(additionalInstances),
		parser.WithAdditionalTaskProperties(request.AdditionalTaskProperties),
	)

	result, err := p.Parse(ctx, strings.Join(yamlConfigs, "\n"))
	if err != nil {
		if re, ok := err.(*parsererror.Rich); ok {
			return &api.EvaluateConfigResponse{
				Error: &api.RichError{
					Message:         re.Message(),
					ProcessedConfig: re.Config(),
					Line:            uint64(re.Line()),
					Column:          uint64(re.Column()),
				},
			}, nil
		}

		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &api.EvaluateConfigResponse{
		Tasks:                     result.Tasks,
		TasksCountBeforeFiltering: result.TasksCountBeforeFiltering,
	}, nil
}

func (r *ConfigurationEvaluatorServiceServer) JSONSchema(
	ctx context.Context,
	request *api.JSONSchemaRequest,
) (*api.JSONSchemaResponse, error) {
	additionalInstances, err := TransformAdditionalInstances(request.AdditionalInstancesInfo)
	if err != nil {
		return nil, err
	}

	// Generate schema
	p := parser.New(parser.WithAdditionalInstances(additionalInstances))

	schemaBytes, err := json.Marshal(p.Schema())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Inject fileMatch field
	var schema map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	schema["fileMatch"] = []string{".cirrus.yml", ".cirrus.yaml"}

	schemaBytes, err = json.Marshal(&schema)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &api.JSONSchemaResponse{Schema: string(schemaBytes)}, nil
}

func (r *ConfigurationEvaluatorServiceServer) EvaluateFunction(
	ctx context.Context,
	request *api.EvaluateFunctionRequest,
) (*api.EvaluateFunctionResponse, error) {
	// Run Starlark hook
	result, err := larker.New().Hook(ctx, request.StarlarkConfig, request.FunctionName, request.Arguments.AsSlice())
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	response := &api.EvaluateFunctionResponse{
		ErrorMessage:  result.ErrorMessage,
		OutputLogs:    result.OutputLogs,
		DurationNanos: result.DurationNanos,
	}

	// Convert Hook()'s interface{} return value to structpb-style value
	resultAsStructpb, err := structpb.NewValue(result.Result)
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("%s() hook returned JSON-incompatible value: %v", request.FunctionName, err)
		return response, nil
	}

	response.Result = resultAsStructpb

	return response, nil
}

func TransformAdditionalInstances(
	additionalInstancesInfo *api.AdditionalInstancesInfo,
) (map[string]protoreflect.MessageDescriptor, error) {
	additionalInstances := make(map[string]protoreflect.MessageDescriptor)

	descriptorSet := additionalInstancesInfo.GetDescriptorSet()
	if descriptorSet == nil {
		return additionalInstances, nil
	}

	files, err := protodesc.NewFiles(descriptorSet)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	for fieldName, fqn := range additionalInstancesInfo.Instances {
		descriptor, err := files.FindDescriptorByName(protoreflect.FullName(fqn))
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to find %s type: %v", fqn, err))
		}

		if md, _ := descriptor.(protoreflect.MessageDescriptor); md != nil {
			additionalInstances[fieldName] = md
		}
	}

	return additionalInstances, nil
}
