package options_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/stretchr/testify/assert"
	"runtime"
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

	// Should be pulled because EagerPull is set to true
	image := canaryImage()

	if err := backend.ImagePull(ctx, image); err != nil {
		t.Fatal(err)
	}

	assert.True(t, do.ShouldPullImage(ctx, backend, image))
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

	// Shouldn't be pulled because it's blacklisted
	assert.False(t, do.ShouldPullImage(ctx, backend, "nonexistent.invalid/should/not/be:pulled"))

	// Should be pulled because it doesn't exist
	assert.True(t, do.ShouldPullImage(ctx, backend, "nonexistent.invalid/some/other:image"))

	// Shouldn't be pulled because it does exist
	image := canaryImage()

	if err := backend.ImagePull(ctx, image); err != nil {
		t.Fatal(err)
	}

	assert.False(t, do.ShouldPullImage(ctx, backend, image))
}

func canaryImage() string {
	result := "debian:latest"

	if runtime.GOOS == "windows" {
		result = "mcr.microsoft.com/windows/servercore:ltsc2019"
	}

	return result
}
