package node_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScalarToString(t *testing.T) {
	yamlSlice := yaml.MapSlice{
		{Key: "boolTrue", Value: true},
		{Key: "boolFalse", Value: false},
		{Key: "string", Value: "some value"},
		{Key: "int", Value: 42},
		{Key: "uint", Value: uint(42)},
		{Key: "float32", Value: float32(3.14159)},
		{Key: "float64", Value: float64(3.14159)},
	}

	tree, err := node.NewFromSlice(yamlSlice)
	if err != nil {
		t.Fatal(err)
	}

	expectedValues := []string{
		"true",
		"false",
		"some value",
		"42",
		"42",
		"3.14159",
		"3.14159",
	}

	for i, expectedValue := range expectedValues {
		assert.Equal(t, expectedValue, tree.Children[i].Value.(*node.ScalarValue).Value)
	}
}
