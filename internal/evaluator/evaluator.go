package evaluator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/version"
	"github.com/cirruslabs/cirrus-cli/pkg/larker"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/failing"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/github"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/memory"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/metric/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
	"net"
	"strings"
)

const pathYAML = ".cirrus.yml"
const pathStarlark = ".cirrus.star"

var ErrNoFS = errors.New("no filesystem available")

type ConfigurationEvaluatorServiceServer struct {
	// must be embedded to have forward compatible implementations
	api.UnimplementedCirrusConfigurationEvaluatorServiceServer
}

func addVersion(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	headers := map[string]string{
		"X-Cirrus-Evaluator-Version": version.FullVersion,
	}

	if err := grpc.SetHeader(ctx, metadata.New(headers)); err != nil {
		return nil, err
	}

	return handler(ctx, req)
}

func Serve(ctx context.Context, lis net.Listener) error {
	server := grpc.NewServer(
		grpc.UnaryInterceptor(addVersion),
		grpc.StatsHandler(otelgrpc.NewServerHandler(
			otelgrpc.WithMeterProvider(noop.NewMeterProvider()),
		)),
	)

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

func (r *ConfigurationEvaluatorServiceServer) EvaluateConfig(
	ctx context.Context,
	request *api.EvaluateConfigRequest,
) (*api.EvaluateConfigResponse, error) {
	result := &api.EvaluateConfigResponse{}

	var yamlConfigs []string

	// Register YAML configuration (if any)
	if request.YamlConfig != "" {
		yamlConfigs = append(yamlConfigs, request.YamlConfig)
	}

	fs, err := convertFS(request.Fs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to initialize file system: %v", err)
	}

	// Run Starlark script and register generated YAML configuration (if any)
	//nolint:nestif // doesn't seem too complicated
	if request.StarlarkConfig != "" {
		lrk := larker.New(
			larker.WithFileSystem(fs),
			larker.WithEnvironment(request.Environment),
			larker.WithAffectedFiles(request.AffectedFiles),
		)

		lrkResult, err := lrk.MainOptional(ctx, request.StarlarkConfig)
		if err == nil {
			result.OutputLogs = lrkResult.OutputLogs

			if lrkResult.YAMLConfig != "" {
				yamlConfigs = append(yamlConfigs, lrkResult.YAMLConfig)
			}
		} else if ee, ok := err.(*larker.ExtendedError); ok {
			result.Issues = append(result.Issues, &api.Issue{
				Level:   api.Issue_ERROR,
				Message: ee.Error(),
				Path:    pathStarlark,
			})
			result.OutputLogs = ee.Logs()
		} else {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
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

	result.ProcessedConfig = strings.Join(yamlConfigs, "\n")

	parseResult, err := p.Parse(ctx, result.ProcessedConfig)
	if err != nil {
		if re, ok := err.(*parsererror.Rich); ok {
			result.Issues = append(result.Issues, &api.Issue{
				Level:   api.Issue_ERROR,
				Message: re.Message(),
				Path:    pathYAML,
				Line:    uint64(re.Line()),
				Column:  uint64(re.Column()),
			})

			return result, nil
		}

		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	result.Tasks = parseResult.Tasks
	result.Issues = append(result.Issues, parseResult.Issues...)
	result.TasksCountBeforeFiltering = parseResult.TasksCountBeforeFiltering

	return result, nil
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
	fs, err := convertFS(request.Fs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to initialize file system: %v", err)
	}

	lrk := larker.New(
		larker.WithFileSystem(fs),
		larker.WithEnvironment(request.Environment),
	)

	// Run Starlark hook
	result, err := lrk.Hook(ctx, request.StarlarkConfig, request.FunctionName, request.Arguments.AsSlice())
	if err != nil {
		if errors.Is(err, larker.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, err.Error())
		}

		if ee, ok := err.(*larker.ExtendedError); ok {
			return &api.EvaluateFunctionResponse{
				ErrorMessage: ee.Error(),
				OutputLogs:   ee.Logs(),
			}, nil
		}

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

func convertFS(apiFS *api.FileSystem) (fs fs.FileSystem, err error) {
	fs = failing.New(ErrNoFS)

	if apiFS == nil {
		return fs, err
	}

	switch impl := apiFS.Impl.(type) {
	case *api.FileSystem_Memory_:
		fs, err = memory.New(impl.Memory.FilesContents)
	case *api.FileSystem_Github_:
		fs, err = github.New(impl.Github.Owner, impl.Github.Repo, impl.Github.Reference, impl.Github.Token)
	}

	return fs, err
}
