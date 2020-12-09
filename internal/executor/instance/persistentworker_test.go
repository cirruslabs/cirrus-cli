package instance_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/stretchr/testify/assert"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestRetrieveAgentBinary(t *testing.T) {
	// Does it work?
	firstPath, err := instance.RetrieveAgentBinary(context.Background(),
		platform.DefaultAgentVersion, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatal()
	}
	firstPathStat, err := os.Stat(firstPath)
	if err != nil {
		t.Fatal(err)
	}

	// Does it cache the agent?
	cachedRetrievalStart := time.Now()

	secondPath, err := instance.RetrieveAgentBinary(context.Background(),
		platform.DefaultAgentVersion, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatal()
	}
	secondPathStat, err := os.Stat(firstPath)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, firstPath, secondPath)
	assert.Equal(t, firstPathStat.ModTime(), secondPathStat.ModTime())
	assert.WithinDuration(t, cachedRetrievalStart, time.Now(), 100*time.Millisecond)
}
