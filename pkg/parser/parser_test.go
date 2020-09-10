package parser_test

import (
	"encoding/json"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
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

func TestSchema(t *testing.T) {
	p := parser.New()

	jsonBytes, err := json.MarshalIndent(p.Schema(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(jsonBytes))
}
