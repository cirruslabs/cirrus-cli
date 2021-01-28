package node_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestDeepFindChild(t *testing.T) {
	yamlSlice := yaml.MapSlice{
		{Key: "task", Value: yaml.MapSlice{
			{Key: "container", Value: yaml.MapSlice{
				{Key: "matrix", Value: yaml.MapSlice{
					{Key: "image", Value: "debian:latest"},
					{Key: "image", Value: "ubuntu:latest"},
				}},
			}},
			{Key: "matrix", Value: yaml.MapSlice{
				{Key: "name", Value: "First task"},
				{Key: "name", Value: "Second task"},
			}},
		}},
	}

	tree, err := node.NewFromSlice(yamlSlice)
	if err != nil {
		t.Fatal(err)
	}

	deepChild := tree.DeepFindChild("matrix")
	require.Len(t, deepChild.Children, 2)
	assert.Equal(t, deepChild.Children[0].Value.(*node.ScalarValue).Value, "debian:latest")
	assert.Equal(t, deepChild.Children[1].Value.(*node.ScalarValue).Value, "ubuntu:latest")
}

func TestDeepFindCollectible(t *testing.T) {
	yamlSlice := yaml.MapSlice{
		// First tree's children
		{Key: "env", Value: yaml.MapSlice{
			{Key: "KEY1", Value: "VALUE1"},
		}},
		// Second tree's children
		{Key: "alpha", Value: yaml.MapSlice{
			{Key: "env", Value: yaml.MapSlice{
				{Key: "KEY2", Value: "VALUE2"},
			}},
		}},
	}

	tree, err := node.NewFromSlice(yamlSlice)
	if err != nil {
		t.Fatal(err)
	}

	virtualNode := tree.Children[1].DeepFindCollectible("env")
	assert.Equal(t, "KEY1", virtualNode.Children[0].Name)
	assert.Equal(t, "KEY2", virtualNode.Children[1].Name)
}

func TestDeepFindChildrenSameLevel(t *testing.T) {
	yamlSlice := yaml.MapSlice{
		{Key: "alpha", Value: yaml.MapSlice{
			{Key: "env", Value: yaml.MapSlice{
				{Key: "KEY1", Value: "VALUE1"},
			}},
			{Key: "env", Value: yaml.MapSlice{
				{Key: "KEY2", Value: "VALUE2"},
			}},
		}},
	}

	tree, err := node.NewFromSlice(yamlSlice)
	if err != nil {
		t.Fatal(err)
	}

	virtualNode := tree.Children[0].DeepFindCollectible("env")
	assert.Len(t, virtualNode.Children, 2)
	assert.Equal(t, "KEY1", virtualNode.Children[0].Name)
	assert.Equal(t, "KEY2", virtualNode.Children[1].Name)
}

func TestHasChildren(t *testing.T) {
	yamlSlice := yaml.MapSlice{
		{Key: "some name", Value: "some value"},
	}

	tree, err := node.NewFromSlice(yamlSlice)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, tree.HasChild("some name"))
}
