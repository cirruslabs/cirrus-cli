package node_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDeepFindChild(t *testing.T) {
	tree, err := node.NewFromText(`task:
  container:
    matrix:
      image: debian:latest
      image: ubuntu:latest

    matrix:
      name: First task
      name: Second task
`)
	if err != nil {
		t.Fatal(err)
	}

	deepChild := tree.DeepFindChild("matrix")
	require.Len(t, deepChild.Children, 2)
	assert.Equal(t, deepChild.Children[0].Value.(*node.ScalarValue).Value, "debian:latest")
	assert.Equal(t, deepChild.Children[1].Value.(*node.ScalarValue).Value, "ubuntu:latest")
}

func TestDeepFindCollectible(t *testing.T) {
	tree, err := node.NewFromText(`env:
  KEY1: VALUE1

alpha:
  env:
    KEY2: VALUE2
`)
	if err != nil {
		t.Fatal(err)
	}

	virtualNode := tree.Children[1].DeepFindCollectible("env")
	assert.Equal(t, "KEY1", virtualNode.Children[0].Name)
	assert.Equal(t, "KEY2", virtualNode.Children[1].Name)
}

func TestDeepFindChildrenSameLevel(t *testing.T) {
	tree, err := node.NewFromText(`alpha:
  env:
    KEY1: VALUE1
  env:
    KEY2: VALUE2
`)
	if err != nil {
		t.Fatal(err)
	}

	virtualNode := tree.Children[0].DeepFindCollectible("env")
	assert.Len(t, virtualNode.Children, 2)
	assert.Equal(t, "KEY1", virtualNode.Children[0].Name)
	assert.Equal(t, "KEY2", virtualNode.Children[1].Name)
}

func TestHasChildren(t *testing.T) {
	tree, err := node.NewFromText(`some name: some value
`)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, tree.HasChild("some name"))
}
