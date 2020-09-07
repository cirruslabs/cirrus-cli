package node_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestDeepFindChildren(t *testing.T) {
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

	virtualNode := tree.Children[1].DeepFindChild("env")
	assert.Equal(t, "KEY2", virtualNode.Children[0].Name)
	assert.Equal(t, "KEY1", virtualNode.Children[1].Name)
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
