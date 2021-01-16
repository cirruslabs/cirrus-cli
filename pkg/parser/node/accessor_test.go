package node_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestGetExpandedStringValue(t *testing.T) {
	yamlSlice := yaml.MapSlice{
		{Key: "name", Value: "Batched $VALUE-${I}"},
	}

	tree, err := node.NewFromSlice(yamlSlice)
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
	yamlSlice := yaml.MapSlice{
		{Key: "env", Value: yaml.MapSlice{
			{Key: "KEY1", Value: "VALUE1"},
			{Key: "KEY2", Value: "VALUE2"},
		}},
	}

	tree, err := node.NewFromSlice(yamlSlice)
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
	yamlSlice := yaml.MapSlice{
		{Key: "script_single_scalar", Value: "command1"},
		{Key: "script_single_list", Value: []interface{}{
			"command1",
		}},
		{Key: "script_multiple_list", Value: []interface{}{
			"command1",
			"command2",
		}},
	}

	tree, err := node.NewFromSlice(yamlSlice)
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
