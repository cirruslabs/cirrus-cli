package node_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFindParent(t *testing.T) {
	tree, err := node.NewFromText(`a:
  b:
    c: 42
    d: 43
`)
	if err != nil {
		t.Fatal(err)
	}

	b := tree.DeepFindChild("b")
	d := tree.DeepFindChild("d")
	assert.Equal(t, b, d.FindParent(func(nodeName string) bool {
		return nodeName == "b"
	}))
	assert.Nil(t, d.FindParent(func(nodeName string) bool {
		return nodeName == "z`"
	}))
}

func TestReversePath(t *testing.T) {
	assert.Equal(t, []int{}, node.ReversePath([]int{}))
	assert.Equal(t, []int{42}, node.ReversePath([]int{42}))
	assert.Equal(t, []int{1, 2, 3}, node.ReversePath([]int{3, 2, 1}))
}

func TestPathUpwardsUpto(t *testing.T) {
	tree, err := node.NewFromText(`a:
  b:
    c: 42
    d: 43
`)
	if err != nil {
		t.Fatal(err)
	}

	d := tree.DeepFindChild("d")
	assert.EqualValues(t, []int{1, 0, 0}, d.PathUpwardsUpto(tree))
}

func TestGetPath(t *testing.T) {
	tree, err := node.NewFromText(`a:
  b:
    c: 42
    d: 43
`)
	if err != nil {
		t.Fatal(err)
	}

	d := tree.DeepFindChild("d")
	assert.Equal(t, d, tree.GetPath([]int{0, 0, 1}))
}

func TestDeepCopy(t *testing.T) {
	tree, err := node.NewFromText(`a:
  b:
    c: 42
    d: 43
`)
	if err != nil {
		t.Fatal(err)
	}

	// Do a copy
	b := tree.DeepFindChild("b")
	bCopy := b.DeepCopy()

	// Ensure that copy's parent is reset
	assert.Nil(t, bCopy.Parent)

	// Ensure that node and it's children were indeed copied
	assert.NotEqual(t, b, bCopy)

	c := tree.DeepFindChild("c")
	cCopy := bCopy.DeepFindChild("c")
	assert.NotNil(t, c)
	assert.NotNil(t, cCopy)
	assert.NotEqual(t, c, cCopy)

	d := tree.DeepFindChild("d")
	dCopy := bCopy.DeepFindChild("d")
	assert.NotNil(t, d)
	assert.NotNil(t, dCopy)
	assert.NotEqual(t, d, dCopy)
}
