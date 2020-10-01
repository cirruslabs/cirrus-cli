package evaluator

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/larker"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"
	"strings"
)

func Serve(ctx context.Context, lis net.Listener) error {
	server := grpc.NewServer()

	api.RegisterCirrusConfigurationEvaluatorServiceService(server, &api.CirrusConfigurationEvaluatorServiceService{
		EvaluateConfig: evaluateConfig,
	})

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

func evaluateConfig(ctx context.Context, request *api.EvaluateConfigRequest) (*api.EvaluateConfigResponse, error) {
	var yamlConfigs []string

	// Register YAML configuration (if any)
	if request.YamlConfig != "" {
		yamlConfigs = append(yamlConfigs, request.YamlConfig)
	}

	// Run Starlark script and register generated YAML configuration (if any)
	if request.StarlarkConfig != "" {
		lrk := larker.New(larker.WithEnvironment(request.Environment))

		generatedYamlConfig, err := lrk.Main(ctx, request.StarlarkConfig)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		yamlConfigs = append(yamlConfigs, generatedYamlConfig)
	}

	// Parse combined YAML
	p := parser.New(
		parser.WithEnvironment(request.Environment),
		parser.WithFilesContents(request.FilesContents),
	)

	result, err := p.Parse(strings.Join(yamlConfigs, "\n"))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if len(result.Errors) != 0 {
		return nil, status.Error(codes.InvalidArgument, result.Errors[0])
	}

	return &api.EvaluateConfigResponse{Tasks: result.Tasks}, nil
}
