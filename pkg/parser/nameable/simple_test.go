package nameable_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSimpleNameable(t *testing.T) {
	assert.True(t, nameable.NewSimpleNameable("").Matches(""))
	assert.True(t, nameable.NewSimpleNameable("a").Matches("a"))
	assert.False(t, nameable.NewSimpleNameable("a").Matches("b"))
}
