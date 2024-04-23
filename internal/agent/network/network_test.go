package network_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/agent/network"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

const (
	maxAllowedWaitTime  = 60 * time.Second
	maxExpectedWaitTime = 10 * time.Second
)

// TestEarlyExit ensures that WaitForLocalPort() will exit before the full waitDuration
// if the port becomes available in the first loop iteration.
func TestEarlyExit(t *testing.T) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()

	port := lis.Addr().(*net.TCPAddr).Port

	ctx, cancel := context.WithTimeout(context.Background(), maxAllowedWaitTime)
	defer cancel()

	start := time.Now()
	network.WaitForLocalPort(ctx, port)
	stop := time.Now()

	assert.WithinDuration(t, stop, start, maxExpectedWaitTime)
}
