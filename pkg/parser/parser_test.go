package parser_test

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"

	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/stretchr/testify/assert"
)

var validCases = []string{
	"example-android.yml",
	"example-flutter-web.yml",
	"example-mysql.yml",
	"example-rust.yml",
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
			result, err := p.ParseFromFile(absolutize(file))

			require.Nil(t, err)
			assert.Empty(t, result.Errors)
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
