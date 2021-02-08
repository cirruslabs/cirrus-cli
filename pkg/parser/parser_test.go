package parser_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/memory"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/cirruslabs/cirrus-cli/pkg/rpcparser"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/stretchr/testify/require"
	"github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v3"
	"io/ioutil"
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
}

var invalidCases = []string{
	"invalid-empty.yml",
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

	// uncomment to update test data
	// ioutil.WriteFile(actualFixturePath, actual, 0600)

	expected, err := ioutil.ReadFile(actualFixturePath)
	if err != nil {
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

func TestInvalidConfigs(t *testing.T) {
	for _, invalidCase := range invalidCases {
		file := invalidCase
		t.Run(file, func(t *testing.T) {
			p := parser.New()
			_, err := p.ParseFromFile(context.Background(), absolutize(file))
			assert.Error(t, err)
		})
	}
}

// TestViaRPC ensures that the parser produces results identical to rpcparser.
func TestViaRPC(t *testing.T) {
	cloudDir := absolutize("via-rpc")

	fileInfos, err := ioutil.ReadDir(cloudDir)
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

	yamlBytes, err := ioutil.ReadFile(yamlConfigPath)
	if err != nil {
		t.Fatal(err)
	}

	// Obtain expected result by loading JSON fixture
	fixtureBytes, err := ioutil.ReadFile(fixturePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			viaRPCCreateJSONFixture(t, yamlBytes, fixturePath, envPath, fcPath)
			t.Fatalf("created new fixture: %s, don't forget to commit it", fixturePath)
		}

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

func viaRPCCreateJSONFixture(t *testing.T, yamlBytes []byte, fixturePath string, envPath string, fcPath string) {
	// Aid in migration by automatically creating new JSON fixture using the RPC parser
	rpcParser := rpcparser.Parser{
		Environment:   viaRPCLoadMap(t, envPath),
		FilesContents: viaRPCLoadMap(t, fcPath),
	}
	rpcResult, err := rpcParser.Parse(string(yamlBytes))
	if err != nil {
		t.Fatal(err)
	}

	fixtureBytes := testutil.TasksToJSON(t, rpcResult.Tasks)
	if err := ioutil.WriteFile(fixturePath, fixtureBytes, 0600); err != nil {
		t.Fatal(err)
	}
}

func viaRPCLoadMap(t *testing.T, yamlPath string) (result map[string]string) {
	yamlBytes, err := ioutil.ReadFile(yamlPath)
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
		{"validation-badDependencies.yml", "error in dependencies between tasks: b, c, d"},
		{"validation-duplicateCommands.yml", "task 'main' cache and script instructions have identical name"},
	}

	for _, testCase := range invalidCases {
		testCase := testCase

		t.Run(testCase.File, func(t *testing.T) {
			yamlBytes, err := ioutil.ReadFile(filepath.Join("testdata", "via-rpc-invalid", testCase.File))
			if err != nil {
				t.Fatal(err)
			}

			localParser := parser.New()
			_, err = localParser.Parse(context.Background(), string(yamlBytes))
			require.Error(t, err, "parser should return an error")
			require.Contains(t, err.Error(), testCase.Message, "parser should return a specific error")
		})
	}
}

func TestSchema(t *testing.T) {
	p := parser.New()

	// Load reference schema
	referenceBytes, err := ioutil.ReadFile("testdata/cirrus.json")
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

	// Remove cloud instances from the reference schema since they're not present in our schema
	delete(referenceObject["patternProperties"].(map[string]interface{}), "^(.*)gke_pipe$")

	ignoredProperties := []string{
		// instances
		"anka_instance",
		"aws_credentials",
		"azure_container_instance",
		"azure_credentials",
		"ec2_instance",
		"eks_container",
		"freebsd_instance",
		"gce_container",
		"gce_instance",
		"gcp_credentials",
		"gke_container",
		"osx_instance",
		"macos_instance",
		// cloud task properties
		"auto_cancellation",
		"execution_lock",
		"experimental",
		"required_pr_labels",
		"skip_notifications",
		"stateful",
		"trigger_type",
		"use_compute_credits",
	}

	for _, ignoredProperty := range ignoredProperties {
		delete(referenceObject["properties"].(map[string]interface{}), ignoredProperty)

		patternedTask := referenceObject["patternProperties"].(map[string]interface{})["^(.*)task$"]
		delete(patternedTask.(map[string]interface{})["properties"].(map[string]interface{}), ignoredProperty)

		patternedDockerBuilder := referenceObject["patternProperties"].(map[string]interface{})["^(.*)docker_builder$"]
		delete(patternedDockerBuilder.(map[string]interface{})["properties"].(map[string]interface{}), ignoredProperty)

		patternedPipe := referenceObject["patternProperties"].(map[string]interface{})["^(.*)pipe$"]
		delete(patternedPipe.(map[string]interface{})["properties"].(map[string]interface{}), ignoredProperty)
	}

	delete(referenceObject, "fileMatch")

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
		{"testdata/rich-errors-pipe.yml", parsererror.NewRich(2, 10,
			"steps should be a list")},
		{"testdata/rich-errors-accessor.yml", parsererror.NewRich(6, 5,
			"expected a scalar value or a list with scalar values")},
		{"testdata/rich-errors-matrix.yml", parsererror.NewRich(4, 7,
			"matrix can be defined only under a task or docker_builder")},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.File, func(t *testing.T) {
			config, err := ioutil.ReadFile(testCase.File)
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
