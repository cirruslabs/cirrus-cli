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

	response, err := evaluateHelper(t, &api.EvaluateConfigRequest{
		YamlConfig:     yamlConfig,
		StarlarkConfig: starlarkConfig,
	})
	require.NoError(t, err)
	require.Len(t, response.Tasks, 4)
}

// TestGitHubFS ensures that evaluator picks up GitHub-related environment variables if present
// and instantiates GitHub filesystem for Starlark execution.
func TestGitHubFS(t *testing.T) {
	starlarkConfig := `
load("cirrus", "fs")

def main(ctx):
    go_mod = fs.read("go.mod")

    if go_mod == None:
        fail("go.mod does not exists")

    canary = "module github.com/cirruslabs/cirrus-cli"

    if canary not in go_mod:
        fail("go.mod does not contain '%s'" % canary)

    return [
        {
            "container": {
                "image": "debian:latest",
            },
            "script": "true",
        },
    ]
`

	env := map[string]string{
		"CIRRUS_REPO_CLONE_TOKEN": "",
		"CIRRUS_REPO_OWNER":       "cirruslabs",
		"CIRRUS_REPO_NAME":        "cirrus-cli",
	}

	// Try specifying a branch
	env["CIRRUS_CHANGE_IN_REPO"] = "master"
	_, err := evaluateHelper(t, &api.EvaluateConfigRequest{
		StarlarkConfig: starlarkConfig,
		Environment:    env,
	})
	require.NoError(t, err)

	// Try specifying a commit currently pointed to by the master branch
	env["CIRRUS_CHANGE_IN_REPO"] = "65368b9c"
	_, err = evaluateHelper(t, &api.EvaluateConfigRequest{
		StarlarkConfig: starlarkConfig,
		Environment:    env,
	})
	require.NoError(t, err)
}
