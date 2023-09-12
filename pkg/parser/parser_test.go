package parser_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/evaluator"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/executorservice"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/memory"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/stretchr/testify/require"
	"github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/stretchr/testify/assert"
)

var validCases = []string{
	"example-android",
	"example-flutter-web",
	"example-mysql",
	"example-rust",
	"instance-persistent_worker",
	"collectible-order",
	"yaml-12-booleans-only",
	"dependency-on-disabled-only-if-task",
	"cache-multiple-folders",
	"no-always-override",
	"pipe-cache",
	"upload-caches",
	"cache-fingerprint-key",
	"docker-arguments-expansion",
	"yaml-scripts-merging",
	"singleGKEContainer3",
	"yaml-merge-sequence",
	"additional-container-port-and-ports-are-optional",
	"yaml-merge-collectible",
	"depends-on-expansion",
	"depends-on-env-list-expansion",
	"persistent-worker-resource-management",
	"arm-container",
}

func absolutize(file string) string {
	return filepath.Join("testdata", file)
}

func TestValidConfigs(t *testing.T) {
	for _, validCase := range validCases {
		file := validCase
		t.Run(file, func(t *testing.T) {
			p := parser.New()
			result, err := p.ParseFromFile(context.Background(), absolutize(file+".yml"))

			require.Nil(t, err)

			assertExpectedTasks(t, absolutize(file+".json"), result)
		})
	}
}

func TestInvalidConfigs(t *testing.T) {
	var invalidCases = []struct {
		Name  string
		Error string
	}{
		{"invalid-missing-required-field", "parsing error: 5:1: required field \"steps\" was not set"},
		{"invalid-upload-caches-nonexistent-cache", "parsing error: 7:3: no cache with name \"mode_nodules\" is defined"},
		{"invalid-cache-two-fingerprints", "parsing error: 5:3: please either use fingerprint_script: or fingerprint_key, " +
			"since otherwise there's ambiguity about which one to prefer for cache key calculation"},
		{"invalid-multiple-name-ambiguity", "parsing error: task 'deploy' depends on task 'rspec_code', which name " +
			"was overridden by a name: field inside of that task"},
		{"invalid-dockerfile-ambiguity", "parsing error: 4:7: container with \"dockerfile:\" also needs a CIRRUS_ARCH" +
			" environment variable to be specified"},
	}

	for _, invalidCase := range invalidCases {
		invalidCase := invalidCase

		t.Run(invalidCase.Name, func(t *testing.T) {
			p := parser.New()
			_, err := p.ParseFromFile(context.Background(), absolutize(invalidCase.Name+".yml"))

			require.Error(t, err)
			assert.Equal(t, invalidCase.Error, err.Error())
		})
	}
}

func TestProblematicConfigs(t *testing.T) {
	var problematicCases = []struct {
		Name           string
		ExpectedIssues []*api.Issue
	}{
		{"problematic-potentially-missed-task", []*api.Issue{
			{Level: api.Issue_WARNING, Message: "you've probably meant foo_task", Path: ".cirrus.yml", Line: 4, Column: 1},
		}},
		{"problematic-artifacts-instruction-is-not-a-map", []*api.Issue{
			{Level: api.Issue_WARNING, Message: "expected a map, found scalar", Path: ".cirrus.yml", Line: 9, Column: 3},
		}},
		{"issues-unbalanced-only-if", []*api.Issue{
			{
				Level:   api.Issue_WARNING,
				Message: "task \"build\" depends on task \"test\", but their only_if conditions are different",
				Path:    ".cirrus.yml",
				Line:    9,
				Column:  1,
			},
		}},
		{"issues-unbalanced-only-if-prevent", []*api.Issue{
			{
				Level:   api.Issue_WARNING,
				Message: "task \"build\" depends on task \"test\", but their only_if conditions are different",
				Path:    ".cirrus.yml",
				Line:    12,
				Column:  1,
			},
		}},
	}

	for _, problematicCase := range problematicCases {
		problematicCase := problematicCase

		t.Run(problematicCase.Name, func(t *testing.T) {
			p := parser.New()
			result, err := p.ParseFromFile(context.Background(), absolutize(problematicCase.Name+".yml"))

			require.NoError(t, err)
			assert.EqualValues(t, problematicCase.ExpectedIssues, result.Issues)
		})
	}
}

func TestAdditionalInstances(t *testing.T) {
	containerInstanceReflect := (&api.ContainerInstance{}).ProtoReflect()
	p := parser.New(parser.WithAdditionalInstances(map[string]protoreflect.MessageDescriptor{
		"proto_container": containerInstanceReflect.Descriptor(),
	}))
	result, err := p.ParseFromFile(context.Background(), absolutize("proto-instance.yml"))

	require.Nil(t, err)
	require.NotEmpty(t, result.Tasks)

	assertExpectedTasks(t, absolutize("proto-instance.json"), result)
}

func TestAdditionalInstancesInvalidProto(t *testing.T) {
	containerInstanceReflect := (&api.ContainerInstance{}).ProtoReflect()
	p := parser.New(parser.WithAdditionalInstances(map[string]protoreflect.MessageDescriptor{
		"proto_container": containerInstanceReflect.Descriptor(),
	}))
	_, err := p.ParseFromFile(context.Background(), absolutize("proto-instance-invalid.yml"))

	require.ErrorContains(t, err, "parsing error: 4:5: failed to find enum value by 'LinusOS' name")
}

func TestAdditionalInstanceDockerfileHashing(t *testing.T) {
	fs, err := memory.New(map[string][]byte{
		"Dockerfile":                            []byte("FROM debian:latest"),
		"Dockerfile.with-arguments":             []byte("FROM debian:latest"),
		"Dockerfile.with-arguments-and-sources": []byte("FROM debian:latest\nCOPY some-file /some-file"),
		"some-file":                             []byte("some contents"),
		"Dockerfile.docker-context":             []byte("FROM debian:latest\nCOPY file /file"),
		"docker-context/file":                   []byte("doesn't matter"),
	})
	if err != nil {
		t.Fatal(err)
	}

	p := parser.New(parser.WithFileSystem(fs), parser.WithAdditionalInstances(map[string]protoreflect.MessageDescriptor{
		"proto_container": (&api.ContainerInstance{}).ProtoReflect().Descriptor(),
	}))
	result, err := p.ParseFromFile(context.Background(), absolutize("additional-instance-dockerfile-hashing.yml"))

	require.Nil(t, err)
	require.NotEmpty(t, result.Tasks)
	require.Empty(t, result.Issues)

	assertExpectedTasks(t, absolutize("additional-instance-dockerfile-hashing.json"), result)
}

func TestAdditionalInstanceDockerfileHashingErrors(t *testing.T) {
	testCases := []struct {
		Name      string
		Config    string
		Files     map[string][]byte
		RichError *parsererror.Rich
	}{
		{
			Name: "nonexistent Dockerfile",
			Config: `task:
  proto_container:
    dockerfile: nonexistent
`,
			Files:     map[string][]byte{},
			RichError: parsererror.NewRich(3, 5, "failed to retrieve \"nonexistent\": file does not exist"),
		},
	}

	additionalInstances := parser.WithAdditionalInstances(map[string]protoreflect.MessageDescriptor{
		"proto_container": (&api.ContainerInstance{}).ProtoReflect().Descriptor(),
	})

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.Name, func(t *testing.T) {
			fs, err := memory.New(testCase.Files)
			if err != nil {
				t.Fatal(err)
			}

			p := parser.New(parser.WithFileSystem(fs), additionalInstances)
			_, err = p.Parse(context.Background(), testCase.Config)

			testCase.RichError.Enrich(testCase.Config)

			assert.Equal(t, testCase.RichError, err)
		})
	}
}

func TestAdditionalInstanceDockerfileHashingWarnings(t *testing.T) {
	fs, err := memory.New(map[string][]byte{
		"Dockerfile": []byte("FORM scratch\n"),
	})
	if err != nil {
		t.Fatal(err)
	}

	additionalInstances := parser.WithAdditionalInstances(map[string]protoreflect.MessageDescriptor{
		"proto_container": (&api.ContainerInstance{}).ProtoReflect().Descriptor(),
	})

	config := `task:
  proto_container:
    dockerfile: Dockerfile
`

	p := parser.New(parser.WithFileSystem(fs), additionalInstances)
	result, err := p.Parse(context.Background(), config)
	require.NoError(t, err)

	expectedIssues := []*api.Issue{
		{
			Level: api.Issue_WARNING,
			Message: "failed to analyze \"Dockerfile\": dockerfile parse error on line 1: " +
				"unknown instruction: FORM (did you mean FROM?)",
			Path:   ".cirrus.yml",
			Line:   3,
			Column: 5,
		},
	}
	require.Equal(t, expectedIssues, result.Issues)
}

func TestAdditionalInstanceStability(t *testing.T) {
	containerInstanceReflect := (&api.ContainerInstance{}).ProtoReflect()
	p := parser.New(parser.WithAdditionalInstances(map[string]protoreflect.MessageDescriptor{
		"red_instance":    containerInstanceReflect.Descriptor(),
		"orange_instance": containerInstanceReflect.Descriptor(),
		"yellow_instance": containerInstanceReflect.Descriptor(),
		"green_instance":  containerInstanceReflect.Descriptor(),
		"blue_instance":   containerInstanceReflect.Descriptor(),
		"purple_instance": containerInstanceReflect.Descriptor(),
	}))
	result, err := p.ParseFromFile(context.Background(), absolutize("additional-instance-stability.yml"))

	require.Nil(t, err)
	require.NotEmpty(t, result.Tasks)

	assertExpectedTasks(t, absolutize("additional-instance-stability.json"), result)
}

func TestCollectiblePropertyOverwrittenByTheUser(t *testing.T) {
	yamlConfig := `windows_container:
  image: mcr.microsoft.com/windows/servercore:ltsc2019

task:
  name: "${CIRRUS_OS}"
  container:
    image: debian:latest
`

	result, err := parser.New().Parse(context.Background(), yamlConfig)

	require.Nil(t, err)
	require.NotEmpty(t, result.Tasks)

	if result.Tasks[0].Name != "linux" {
		t.Fatal("CIRRUS_OS should expand to \"linux\"")
	}
}

func TestAdditionalTaskProperties(t *testing.T) {
	protoName := "custom_bool"
	protoType := descriptor.FieldDescriptorProto_Type(8)
	p := parser.New(parser.WithAdditionalTaskProperties([]*descriptor.FieldDescriptorProto{
		{
			Name: &protoName,
			Type: &protoType,
		},
	}))
	result, err := p.ParseFromFile(context.Background(), absolutize("proto-task-properties.yml"))

	require.Nil(t, err)
	require.NotEmpty(t, result.Tasks)

	assertExpectedTasks(t, absolutize("proto-task-properties.json"), result)
}

func assertExpectedTasks(t *testing.T, actualFixturePath string, result *parser.Result) {
	actual := testutil.TasksToJSON(t, result.Tasks)

	expected, err := os.ReadFile(actualFixturePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(actualFixturePath, actual, 0600); err != nil {
				t.Fatal(err)
			}

			t.Fatalf("created a new fixture %s, don't forget to commit it!\n", actualFixturePath)
		}

		t.Fatal(err)
	}

	// Compare two schemas
	var referenceArray []interface{}
	if err := json.Unmarshal(expected, &referenceArray); err != nil {
		t.Fatal(err)
	}

	var ourArray []interface{}
	if err := json.Unmarshal(actual, &ourArray); err != nil {
		t.Fatal(err)
	}

	differ := gojsondiff.New()
	d := differ.CompareArrays(referenceArray, ourArray)

	if d.Modified() {
		var diffString string

		config := formatter.AsciiFormatterConfig{
			ShowArrayIndex: true,
			Coloring:       true,
		}

		diffString, err = formatter.NewAsciiFormatter(referenceArray, config).Format(d)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Print(diffString)

		t.Fail()
	}
}

// TestViaRPC ensures that the parser produces results identical to the now removed RPC parser.
func TestViaRPC(t *testing.T) {
	cloudDir := absolutize("via-rpc")

	fileInfos, err := os.ReadDir(cloudDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, fileInfo := range fileInfos {
		fileInfo := fileInfo

		if !strings.HasSuffix(fileInfo.Name(), ".yml") {
			continue
		}

		t.Run(fileInfo.Name(), func(t *testing.T) {
			viaRPCRunSingle(t, cloudDir, fileInfo.Name())
		})
	}
}

func viaRPCRunSingle(t *testing.T, cloudDir string, yamlConfigName string) {
	baseName := strings.TrimSuffix(yamlConfigName, filepath.Ext(yamlConfigName))
	yamlConfigPath := filepath.Join(cloudDir, yamlConfigName)
	fixturePath := filepath.Join(cloudDir, baseName+".json")
	envPath := filepath.Join(cloudDir, baseName+".env")
	fcPath := filepath.Join(cloudDir, baseName+".fc")

	yamlBytes, err := os.ReadFile(yamlConfigPath)
	if err != nil {
		t.Fatal(err)
	}

	// Craft virtual in-memory filesystem with test-specific files
	fileContents := make(map[string][]byte)

	for key, value := range viaRPCLoadMap(t, fcPath) {
		fileContents[key] = []byte(value)
	}

	fs, err := memory.New(fileContents)
	if err != nil {
		t.Fatal(err)
	}

	// Obtain the actual result by parsing YAML configuration using the local parser
	localParser := parser.New(
		parser.WithEnvironment(viaRPCLoadMap(t, envPath)),
		parser.WithFileSystem(fs),
	)
	localResult, err := localParser.Parse(context.Background(), string(yamlBytes))
	if err != nil {
		t.Fatal(err)
	}

	// Obtain expected result by loading JSON fixture
	fixtureBytes, err := os.ReadFile(fixturePath)
	if os.IsNotExist(err) {
		fixtureBytes = testutil.TasksToJSON(t, localResult.Tasks)
		err := os.WriteFile(fixturePath, fixtureBytes, 0600)
		if err != nil {
			t.Fatal(err)
		}
	} else if err != nil {
		t.Fatal(err)
	}

	// Compare two schemas
	var referenceArray []interface{}
	if err := json.Unmarshal(fixtureBytes, &referenceArray); err != nil {
		t.Fatal(err)
	}

	var ourArray []interface{}
	if err := json.Unmarshal(testutil.TasksToJSON(t, localResult.Tasks), &ourArray); err != nil {
		t.Fatal(err)
	}

	differ := gojsondiff.New()
	d := differ.CompareArrays(referenceArray, ourArray)

	if d.Modified() {
		var diffString string

		config := formatter.AsciiFormatterConfig{
			ShowArrayIndex: true,
			Coloring:       true,
		}

		diffString, err = formatter.NewAsciiFormatter(referenceArray, config).Format(d)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Print(diffString)

		t.Fail()
	}
}

func viaRPCLoadMap(t *testing.T, yamlPath string) (result map[string]string) {
	yamlBytes, err := os.ReadFile(yamlPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}
		}

		t.Fatal(err)
	}

	if err := yaml.Unmarshal(yamlBytes, &result); err != nil {
		t.Fatal(err)
	}

	return
}

func TestViaRPCInvalid(t *testing.T) {
	invalidCases := []struct {
		File    string
		Message string
	}{
		{
			"validation-badDependencies.yml",
			"parsing error: error in dependencies between tasks: b, c, d",
		},
		{
			"validation-duplicateCommands.yml",
			"parsing error: task 'main' has cache and script instructions with an identical name 'foo'",
		},
		{
			"validation-missingDependency.yml",
			"parsing error: there's no task 'fooo', but task 'bar' depends on it",
		},
	}

	for _, testCase := range invalidCases {
		testCase := testCase

		t.Run(testCase.File, func(t *testing.T) {
			yamlBytes, err := os.ReadFile(filepath.Join("testdata", "via-rpc-invalid", testCase.File))
			if err != nil {
				t.Fatal(err)
			}

			localParser := parser.New()
			_, err = localParser.Parse(context.Background(), string(yamlBytes))
			require.Error(t, err, "parser should return an error")
			require.Equal(t, testCase.Message, err.Error(), "parser should return a specific error")
		})
	}
}

func TestSchema(t *testing.T) {
	p := parser.New()

	const schemaPath = "testdata/cirrus.json"

	// Load reference schema
	referenceBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	var referenceObject map[string]interface{}
	if err := json.Unmarshal(referenceBytes, &referenceObject); err != nil {
		t.Fatal(err)
	}

	// Load our schema
	ourBytes, err := json.MarshalIndent(p.Schema(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	var ourObject map[string]interface{}
	if err := json.Unmarshal(ourBytes, &ourObject); err != nil {
		t.Fatal(err)
	}

	// Uncomment to update schema
	// if err := os.WriteFile(schemaPath, ourBytes, 0600); err != nil {
	//	t.Fatal(err)
	// }

	// Compare two schemas
	differ := gojsondiff.New()
	d := differ.CompareObjects(referenceObject, ourObject)

	if d.Modified() {
		var diffString string

		config := formatter.AsciiFormatterConfig{
			ShowArrayIndex: true,
			Coloring:       true,
		}

		diffString, err = formatter.NewAsciiFormatter(referenceObject, config).Format(d)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Print(diffString)

		t.Fail()
	}
}

func TestTasksCountBeforeFiltering(t *testing.T) {
	p := parser.New()
	result, err := p.ParseFromFile(context.Background(), "testdata/tasks-count-before-filtering.yml")
	if err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, 2, result.TasksCountBeforeFiltering)
}

func TestRichErrors(t *testing.T) {
	testCases := []struct {
		File  string
		Error *parsererror.Rich
	}{
		{"testdata/rich-errors-pipe.yml", parsererror.NewRich(2, 3,
			"steps should be a list")},
		{"testdata/rich-errors-accessor.yml", parsererror.NewRich(5, 3,
			"expected a scalar value or a list with scalar values")},
		{"testdata/rich-errors-matrix.yml", parsererror.NewRich(3, 5,
			"matrix can be defined only under a task, docker_builder or pipe")},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.File, func(t *testing.T) {
			config, err := os.ReadFile(testCase.File)
			if err != nil {
				t.Fatal(err)
			}

			testCase.Error.Enrich(string(config))

			p := parser.New()
			_, err = p.Parse(context.Background(), string(config))
			assert.Error(t, err)
			assert.Equal(t, testCase.Error, err)
		})
	}
}

func TestWithMissingInstancesAllowed(t *testing.T) {
	config, err := os.ReadFile("testdata/missing-instances.yml")
	if err != nil {
		t.Fatal(err)
	}

	p := parser.New(parser.WithMissingInstancesAllowed())
	result, err := p.Parse(context.Background(), string(config))
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, result.Tasks, 1)
	assert.Nil(t, result.Tasks[0].Instance)
}

var repeatedKeysYAML = `
container: {image: 'debian:latest'}
task: {name: task1, script: true}
task: {name: task2, script: true}
task: {name: task3, script: true}
`

func TestRepeatedKeys(t *testing.T) {
	p := parser.New()
	result, err := p.Parse(context.Background(), repeatedKeysYAML)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, result.Tasks, 3)
}

func TestAdditionalInstanceCredentials(t *testing.T) {
	supportedInstances, err := executorservice.New().SupportedInstances()
	require.NoError(t, err)
	additionalInstances, err := evaluator.TransformAdditionalInstances(supportedInstances)
	require.NoError(t, err)

	p := parser.New(parser.WithAdditionalInstances(additionalInstances))
	result, err := p.ParseFromFile(context.Background(),
		absolutize("additional-instance-credentials.yml"))
	require.Nil(t, err)
	require.Len(t, result.Tasks, 1)
	assertExpectedTasks(t, absolutize("additional-instance-credentials.json"), result)

	p = parser.New(parser.WithAdditionalInstances(additionalInstances))
	resultAsMap, err := p.ParseFromFile(context.Background(),
		absolutize("additional-instance-credentials-as-map.yml"))
	require.Nil(t, err)
	require.Len(t, resultAsMap.Tasks, 1)
	assertExpectedTasks(t, absolutize("additional-instance-credentials-as-map.json"), resultAsMap)

	// Instances should be identical
	instance, err := anypb.UnmarshalNew(result.Tasks[0].Instance, proto.UnmarshalOptions{})
	require.NoError(t, err)

	instanceAsMap, err := anypb.UnmarshalNew(resultAsMap.Tasks[0].Instance, proto.UnmarshalOptions{})
	require.NoError(t, err)

	require.EqualValues(t, instance.(*dynamicpb.Message).String(), instanceAsMap.(*dynamicpb.Message).String())
}
