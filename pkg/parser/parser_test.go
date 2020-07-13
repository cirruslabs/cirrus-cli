package parser_test

import (
	"errors"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"

	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

const fakeGraphqlEndpoint = "https://api.invalid"

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
	return filepath.Join("..", "..", "testdata", "parser", file)
}

func TestValidConfigs(t *testing.T) {
	for _, validCase := range validCases {
		file := validCase
		t.Run(file, func(t *testing.T) {
			p := parser.Parser{}
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
			p := parser.Parser{}
			result, err := p.ParseFromFile(absolutize(file))

			require.Nil(t, err)
			assert.NotEmpty(t, result.Errors)
		})
	}
}

// TestErrTransport ensures that network-related errors result in ErrTransport.
func TestErrTransport(t *testing.T) {
	gock.New(fakeGraphqlEndpoint).Reply(500)
	defer gock.Off()

	p := parser.Parser{GraphqlEndpoint: fakeGraphqlEndpoint}
	result, err := p.Parse("a: b")

	assert.Nil(t, result)
	assert.True(t, errors.Is(err, parser.ErrTransport))
}

// TestErrInternal ensures that unexpected errors result in ErrInternal.
func TestErrInternal(t *testing.T) {
	gock.New(fakeGraphqlEndpoint).Reply(200).BodyString("{}")
	defer gock.Off()

	p := parser.Parser{GraphqlEndpoint: fakeGraphqlEndpoint}
	result, err := p.Parse("a: b")

	assert.Nil(t, result)
	assert.True(t, errors.Is(err, parser.ErrInternal))
}
