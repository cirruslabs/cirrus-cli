package parser_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/memory"
	"github.com/cirruslabs/cirrus-cli/pkg/rpcparser"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v2"
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
			require.Empty(t, result.Errors)

			expected, err := ioutil.ReadFile(absolutize(file + ".json"))
			if err != nil {
				t.Fatal(err)
			}

			actual := testutil.TasksToJSON(t, result.Tasks)

			assert.JSONEq(t, string(expected), string(actual))
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
	require.Empty(t, result.Errors)
	require.NotEmpty(t, result.Tasks)

	expected, err := ioutil.ReadFile(absolutize("proto-instance.json"))
	if err != nil {
		t.Fatal(err)
	}

	actual := testutil.TasksToJSON(t, result.Tasks)

	assert.JSONEq(t, string(expected), string(actual))
}

func TestInvalidConfigs(t *testing.T) {
	for _, invalidCase := range invalidCases {
		file := invalidCase
		t.Run(file, func(t *testing.T) {
			p := parser.New()
			result, err := p.ParseFromFile(context.Background(), absolutize(file))

			require.Nil(t, err)
			assert.NotEmpty(t, result.Errors)
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
	if len(localResult.Errors) != 0 {
		t.Fatal(localResult.Errors)
	}

	assert.JSONEq(t, string(fixtureBytes), string(testutil.TasksToJSON(t, localResult.Tasks)))
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
	if len(rpcResult.Errors) != 0 {
		t.Fatal(rpcResult.Errors)
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

func TestSchema(t *testing.T) {
	p := parser.New()

	jsonBytes, err := json.MarshalIndent(p.Schema(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(jsonBytes))
}
