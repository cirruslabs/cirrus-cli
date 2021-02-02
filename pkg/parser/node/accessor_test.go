package node_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"testing"
)

func YamlNodeFromString(t *testing.T, text string) *yaml.Node {
	var result yaml.Node

	if err := yaml.Unmarshal([]byte(text), &result); err != nil {
		t.Fatal(err)
	}

	return &result
}

func TestGetExpandedStringValue(t *testing.T) {
	tree, err := node.NewFromNode(YamlNodeFromString(t, `name: Batched $VALUE-${I}
`))
	if err != nil {
		t.Fatal(err)
	}

	env := map[string]string{
		"VALUE": "task",
		"I":     "0",
	}

	expanded, err := tree.Children[0].GetExpandedStringValue(env)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Batched task-0", expanded)
}

func TestGetStringMapping(t *testing.T) {
	tree, err := node.NewFromNode(YamlNodeFromString(t, `env:
  KEY1: VALUE1
  KEY2: VALUE2
`))
	if err != nil {
		t.Fatal(err)
	}

	mapping, err := tree.Children[0].GetStringMapping()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, map[string]string{
		"KEY1": "VALUE1",
		"KEY2": "VALUE2",
	}, mapping)
}

func TestGetSliceOfNonEmptyStrings(t *testing.T) {
	tree, err := node.NewFromNode(YamlNodeFromString(t, `script_single_scalar: command1
script_single_list:
  - command1
script_multiple_list:
  - command1
  - command2
`))
	if err != nil {
		t.Fatal(err)
	}

	slice, err := tree.Children[0].GetSliceOfNonEmptyStrings()
	require.NoError(t, err)
	assert.Equal(t, []string{"command1"}, slice)

	slice, err = tree.Children[1].GetSliceOfNonEmptyStrings()
	require.NoError(t, err)
	assert.Equal(t, []string{"command1"}, slice)

	slice, err = tree.Children[2].GetSliceOfNonEmptyStrings()
	require.NoError(t, err)
	assert.Equal(t, []string{"command1", "command2"}, slice)
}
