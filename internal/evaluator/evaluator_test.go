package evaluator_test

import (
	"context"
	"errors"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/evaluator"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"net"
	"testing"
)

func evaluateHelper(t *testing.T, request *api.EvaluateConfigRequest) (*api.EvaluateConfigResponse, error) {
	ctx, cancel := context.WithCancel(context.Background())

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	errChan := make(chan error)

	go func() {
		errChan <- evaluator.Serve(ctx, lis)
	}()

	defer func() {
		cancel()

		if err := <-errChan; err != nil && !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewCirrusConfigurationEvaluatorServiceClient(conn)

	response, err := client.EvaluateConfig(context.Background(), request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// TestCrossDependencies ensures that tasks declared in YAML and generated from Starlark can reference each other.
func TestCrossDependencies(t *testing.T) {
	yamlConfig := `
container:
  image: debian:latest

task:
  name: Black
  depends_on:
    - White
  script: true

task:
  name: Green
  script: true
`

	starlarkConfig := `
def main(ctx):
    return [
        {
            "name": "White",
            "script": "true",
        },
        {
            "name": "Yellow",
            "depends_on": [
                "Green",
            ],
            "script": "true",
        }
    ]
`

	_, err := evaluateHelper(t, &api.EvaluateConfigRequest{
		YamlConfig:     yamlConfig,
		StarlarkConfig: starlarkConfig,
	})
	require.NoError(t, err)
}
