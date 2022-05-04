//go:build !windows
// +build !windows

package evaluator_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/evaluator"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"net"
	"testing"
	"time"
)

func getClient(t *testing.T) api.CirrusConfigurationEvaluatorServiceClient {
	ctx, cancel := context.WithCancel(context.Background())

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	errChan := make(chan error)

	go func() {
		errChan <- evaluator.Serve(ctx, lis)
	}()

	t.Cleanup(func() {
		cancel()

		if err := <-errChan; err != nil && !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	})

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}

	return api.NewCirrusConfigurationEvaluatorServiceClient(conn)
}

func evaluateConfigHelper(t *testing.T, request *api.EvaluateConfigRequest) (*api.EvaluateConfigResponse, error) {
	return getClient(t).EvaluateConfig(context.Background(), request)
}

func schemaHelper(t *testing.T, request *api.JSONSchemaRequest) (*api.JSONSchemaResponse, error) {
	return getClient(t).JSONSchema(context.Background(), request)
}

func evaluateFunctionHelper(t *testing.T, request *api.EvaluateFunctionRequest) (*api.EvaluateFunctionResponse, error) {
	return getClient(t).EvaluateFunction(context.Background(), request)
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

	response, err := evaluateConfigHelper(t, &api.EvaluateConfigRequest{
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
	_, err := evaluateConfigHelper(t, &api.EvaluateConfigRequest{
		StarlarkConfig: starlarkConfig,
		Environment:    env,
	})
	require.NoError(t, err)

	// Try specifying a commit currently pointed to by the master branch
	env["CIRRUS_CHANGE_IN_REPO"] = "65368b9c"
	_, err = evaluateConfigHelper(t, &api.EvaluateConfigRequest{
		StarlarkConfig: starlarkConfig,
		Environment:    env,
	})
	require.NoError(t, err)
}

// TestAdditionalInstances ensures that dynamically provided instances are respected.
func TestAdditionalContainerInstances(t *testing.T) {
	yamlConfig := `
aliases: &container_body
  image: alpine:latest
  platform: linux
  cpu: 2.5
  memory: 4G
  additional_containers:
    - name: mysql
      image: mysql:latest
      cpu: 1
      memory: 1024
      port: 3306
      environment:
        MYSQL_ROOT_PASSWORD: ""

regular_task:
  container:
    <<: *container_body

proto_task:
  proto_container:
    <<: *container_body
`

	evaluateTwoTasksIdentical(t, yamlConfig, map[string]string{
		"proto_container": "org.cirruslabs.ci.services.cirruscigrpc.ContainerInstance",
	})
}

// TestAdditionalInstances ensures that complex dynamically provided instances are respected.
func TestAdditionalWorkersInstances(t *testing.T) {
	yamlConfig := `
aliases: &persistent_worker
  isolation:
    parallels:
      image: big-sur
      user: admin
      password: admin
      platform: darwin

regular_task:
  persistent_worker:
    <<: *persistent_worker

proto_task:
  proto_persistent_worker:
    <<: *persistent_worker
`

	evaluateTwoTasksIdentical(t, yamlConfig, map[string]string{
		"proto_persistent_worker": "org.cirruslabs.ci.services.cirruscigrpc.PersistentWorkerInstance",
	})
}

// TestAdditionalInstances ensures that complex dynamically provided instances are respected.
func TestAdditionalWorkersEmptyInstances(t *testing.T) {
	yamlConfig := `
regular_task:
  persistent_worker: {}

proto_task:
  proto_persistent_worker: {}
`

	evaluateTwoTasksIdentical(t, yamlConfig, map[string]string{
		"proto_persistent_worker": "org.cirruslabs.ci.services.cirruscigrpc.PersistentWorkerInstance",
	})
}

func evaluateTwoTasksIdentical(t *testing.T, yamlConfig string, additionalInstancesMapping map[string]string) {
	response, err := evaluateConfigHelper(t, &api.EvaluateConfigRequest{
		YamlConfig: yamlConfig,
		AdditionalInstancesInfo: &api.AdditionalInstancesInfo{
			Instances: additionalInstancesMapping,
			DescriptorSet: &descriptor.FileDescriptorSet{
				File: []*descriptor.FileDescriptorProto{
					protodesc.ToFileDescriptorProto(api.File_cirrus_ci_service_proto),
					protodesc.ToFileDescriptorProto(anypb.File_google_protobuf_any_proto),
					protodesc.ToFileDescriptorProto(emptypb.File_google_protobuf_empty_proto),
					protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto),
					protodesc.ToFileDescriptorProto(structpb.File_google_protobuf_struct_proto),
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, response.Tasks, 2)
	require.JSONEq(t, protojson.Format(response.Tasks[0].Instance), protojson.Format(response.Tasks[1].Instance))
	require.Equal(t, response.Tasks[0].Environment, response.Tasks[1].Environment)
}

func TestSchemaHasFileMatch(t *testing.T) {
	response, err := schemaHelper(t, &api.JSONSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}

	var schemaObject map[string]interface{}
	if err := json.Unmarshal([]byte(response.Schema), &schemaObject); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []interface{}{".cirrus.yml", ".cirrus.yaml"}, schemaObject["fileMatch"])
}

func TestSchemaValid(t *testing.T) {
	response, err := schemaHelper(t, &api.JSONSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}

	loader := gojsonschema.NewStringLoader(response.Schema)
	if _, err := gojsonschema.NewSchema(loader); err != nil {
		t.Fatal(err)
	}
}

func TestAffectedFiles(t *testing.T) {
	yamlConfig := `
container:
  image: debian:latest

task:
  only_if: "changesInclude('.cirrus.yml', 'dev/**', 'bin/**') && $CIRRUS_PR != ''"
  script: true
`

	response, err := evaluateConfigHelper(t, &api.EvaluateConfigRequest{
		YamlConfig:    yamlConfig,
		AffectedFiles: []string{"bin/internal/external"},
		Environment:   map[string]string{"CIRRUS_PR": "1234"},
	})
	require.NoError(t, err)
	require.Len(t, response.Tasks, 1)
}

func TestRichErrors(t *testing.T) {
	yamlConfig := `container:
  image:
    should_be: a_string

task:
  script: true
`

	response, err := evaluateConfigHelper(t, &api.EvaluateConfigRequest{YamlConfig: yamlConfig})
	require.NoError(t, err)
	require.NotNil(t, response.Issues)

	firstIssue := response.Issues[0]
	assert.NotEmpty(t, response.ProcessedConfig)
	assert.EqualValues(t, api.Issue_ERROR, firstIssue.Level)
	assert.EqualValues(t, firstIssue.Message, "not a scalar value")
	assert.EqualValues(t, firstIssue.Line, 2)
	assert.EqualValues(t, firstIssue.Column, 3)
}

func TestHook(t *testing.T) {
	config := `load("cirrus", "env")

def on_build_failure(ctx):
  print(env.get("SOME_VARIABLE"))
  return [ctx.build.id, ctx.task.id]
`

	arguments, err := structpb.NewList([]interface{}{
		map[string]interface{}{
			"build": map[string]interface{}{
				"id": 42,
			},
			"task": map[string]interface{}{
				"id": 43,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := evaluateFunctionHelper(t, &api.EvaluateFunctionRequest{
		StarlarkConfig: config,
		FunctionName:   "on_build_failure",
		Arguments:      arguments,
		Environment:    map[string]string{"SOME_VARIABLE": "some value"},
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := []interface{}{
		42,
		43,
	}
	expectedStructpb, err := structpb.NewValue(expected)
	if err != nil {
		t.Fatal(err)
	}

	assert.Empty(t, res.ErrorMessage, "hook should evaluate successfully")

	assert.Equal(t, string(res.OutputLogs), "some value\n", "hook should generate some debugging output")

	assert.Greater(t, res.DurationNanos, int64(0),
		"execution time doesn't seem to be counted properly")
	assert.Less(t, res.DurationNanos, (time.Millisecond * 300).Nanoseconds(),
		"execution time doesn't seem to be counted properly")

	assert.Equal(t, expectedStructpb, res.Result, "hook should return a list of build and task IDs")
}

func TestStarlarkOutputLogs(t *testing.T) {
	starlarkConfig := `def main(ctx):
    print("Foo")
    print("Bar")

    return []
`

	response, err := evaluateConfigHelper(t, &api.EvaluateConfigRequest{StarlarkConfig: starlarkConfig})
	require.NoError(t, err)
	require.Equal(t, "Foo\nBar\n", string(response.OutputLogs))
}

func TestStarlarkNoMain(t *testing.T) {
	starlarkConfig := `def foo():
    return []
`

	response, err := evaluateConfigHelper(t, &api.EvaluateConfigRequest{StarlarkConfig: starlarkConfig})
	require.NoError(t, err)
	require.Empty(t, response.ProcessedConfig)
}

func TestBacktraceMain(t *testing.T) {
	starlarkConfig := `def main(ctx):
    print("main")
    print("sentinel")

    a = []
    return a[0]
`

	response, err := evaluateConfigHelper(t, &api.EvaluateConfigRequest{StarlarkConfig: starlarkConfig})
	require.NoError(t, err)
	require.Contains(t, string(response.OutputLogs), "main\nsentinel\n")
	require.Contains(t, string(response.OutputLogs), "Traceback (most recent call last)")

	for _, issue := range response.Issues {
		if issue.Path != ".cirrus.star" {
			t.Error("there should be only Starlark-specific issues in this test")
		}
	}
}

func TestBacktraceHook(t *testing.T) {
	starlarkConfig := `def on_build_failure(ctx):
  print("hook")
  print("sentinel")

  a = []
  print(a[0])
`

	arguments, err := structpb.NewList([]interface{}{
		map[string]interface{}{
			"build": map[string]interface{}{
				"id": 42,
			},
			"task": map[string]interface{}{
				"id": 43,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	response, err := evaluateFunctionHelper(t, &api.EvaluateFunctionRequest{
		StarlarkConfig: starlarkConfig,
		FunctionName:   "on_build_failure",
		Arguments:      arguments,
	})
	require.NoError(t, err)
	require.Contains(t, string(response.OutputLogs), "hook\nsentinel\n")
	require.Contains(t, string(response.OutputLogs), "Traceback (most recent call last):\n  .cirrus.star:6:10")
}
