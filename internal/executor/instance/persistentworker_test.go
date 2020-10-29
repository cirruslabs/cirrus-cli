package instance_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
	"time"
)

func TestRetrieveAgentBinary(t *testing.T) {
	// Does it work?
	_, err := instance.RetrieveAgentBinary(context.Background(), instance.AgentVersion, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatal()
	}

	// Does it cache the agent?
	cachedRetrievalStart := time.Now()

	_, err = instance.RetrieveAgentBinary(context.Background(), instance.AgentVersion, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatal()
	}

	assert.WithinDuration(t, cachedRetrievalStart, time.Now(), 100*time.Millisecond)
}
