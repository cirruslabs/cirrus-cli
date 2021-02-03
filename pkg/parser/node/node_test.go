package node_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScalarToString(t *testing.T) {
	tree, err := node.NewFromText(`boolTrue: true
boolFalse: false
string: "some value"
integer: 42
float: 3.14159
`)
	if err != nil {
		t.Fatal(err)
	}

	expectedValues := []string{
		"true",
		"false",
		"some value",
		"42",
		"3.14159",
	}

	for i, expectedValue := range expectedValues {
		assert.Equal(t, expectedValue, tree.Children[i].Value.(*node.ScalarValue).Value)
	}
}
