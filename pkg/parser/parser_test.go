package parser_test

import (
	"encoding/json"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/rpcparser"
	"github.com/go-test/deep"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"path/filepath"
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
			result, err := p.ParseFromFile(absolutize(file + ".yml"))

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

func TestInvalidConfigs(t *testing.T) {
	for _, invalidCase := range invalidCases {
		file := invalidCase
		t.Run(file, func(t *testing.T) {
			p := parser.New()
			result, err := p.ParseFromFile(absolutize(file))

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

		t.Run(fileInfo.Name(), func(t *testing.T) {
			yamlConfigPath := filepath.Join(cloudDir, fileInfo.Name())

			rpcParser := rpcparser.Parser{}
			rpcResult, err := rpcParser.ParseFromFile(yamlConfigPath)
			if err != nil {
				t.Fatal(err)
			}
			if len(rpcResult.Errors) != 0 {
				t.Fatal(rpcResult.Errors)
			}

			localParser := parser.New()
			localResult, err := localParser.ParseFromFile(yamlConfigPath)
			if err != nil {
				t.Fatal(err)
			}
			if len(localResult.Errors) != 0 {
				t.Fatal(localResult.Errors)
			}

			differences := deep.Equal(rpcResult.Tasks, localResult.Tasks)
			for _, difference := range differences {
				fmt.Println(difference)
			}
			if len(differences) != 0 {
				t.Fatal("found differences")
			}
		})
	}
}

func TestSchema(t *testing.T) {
	p := parser.New()

	jsonBytes, err := json.MarshalIndent(p.Schema(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(jsonBytes))
}
