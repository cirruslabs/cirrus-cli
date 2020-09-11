package options_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestShouldPullImagePositive(t *testing.T) {
	do := options.DockerOptions{
		NoPullImages: []string{"nonexistent.invalid/should/not/be:pulled"},
	}

	assert.False(t, do.ShouldPullImage("nonexistent.invalid/should/not/be:pulled"))
	assert.True(t, do.ShouldPullImage("nonexistent.invalid/some/other:image"))
}

func TestShouldPullImageNegative(t *testing.T) {
	do := options.DockerOptions{
		NoPull:       true,
		NoPullImages: []string{"nonexistent.invalid/should/not/be:pulled"},
	}

	assert.False(t, do.ShouldPullImage("nonexistent.invalid/should/not/be:pulled"))
	assert.False(t, do.ShouldPullImage("nonexistent.invalid/some/other:image"))
}
