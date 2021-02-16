package options_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestForcePull(t *testing.T) {
	do := options.ContainerOptions{
		EagerPull:    true,
		NoPullImages: []string{"nonexistent.invalid/should/not/be:pulled"},
	}

	ctx := context.Background()
	backend, err := containerbackend.New(containerbackend.BackendAuto)
	if err != nil {
		t.Fatal(err)
	}

	// Shouldn't be pulled because it's blacklisted
	assert.False(t, do.ShouldPullImage(ctx, backend, "nonexistent.invalid/should/not/be:pulled"))

	if err := backend.ImagePull(ctx, "debian:latest"); err != nil {
		t.Fatal(err)
	}

	// Should be pulled because EagerPull is set to true
	assert.True(t, do.ShouldPullImage(ctx, backend, "debian:latest"))
}

func TestNormalPull(t *testing.T) {
	do := options.ContainerOptions{
		NoPullImages: []string{"nonexistent.invalid/should/not/be:pulled"},
	}

	ctx := context.Background()
	backend, err := containerbackend.New(containerbackend.BackendAuto)
	if err != nil {
		t.Fatal(err)
	}

	if err := backend.ImagePull(ctx, "debian:latest"); err != nil {
		t.Fatal(err)
	}

	// Shouldn't be pulled because it's blacklisted
	assert.False(t, do.ShouldPullImage(ctx, backend, "nonexistent.invalid/should/not/be:pulled"))

	// Should be pulled because it doesn't exist
	assert.True(t, do.ShouldPullImage(ctx, backend, "nonexistent.invalid/some/other:image"))

	// Shouldn't be pulled because it does exist
	assert.False(t, do.ShouldPullImage(ctx, backend, "debian:latest"))
}
