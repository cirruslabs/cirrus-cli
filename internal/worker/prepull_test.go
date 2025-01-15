package worker_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/worker"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNeedsPrePullDefaultConfiguration(t *testing.T) {
	// When LastCheck is not set, NeedsPrePull() should always be true
	require.True(t, worker.TartPrePull{}.NeedsPrePull())
}

func TestNeedsPrePullCustomConfiguration(t *testing.T) {
	prePull := worker.TartPrePull{
		CheckInterval: 3 * time.Second,
		Jitter:        3 * time.Second,
	}

	// Since LastCheck defaults to "January 1, year 1, 00:00:00.000000000 UTC",
	// the result of the first call to NeedsPrePull() should always be true
	require.True(t, prePull.NeedsPrePull())

	// Let's imagine that we've actually performed the pre-pull check
	prePull.LastCheck = time.Now()

	// Right after performing the check the result of NeedsPrePull() should be false
	require.False(t, prePull.NeedsPrePull())

	// Now we're likely still in the CheckInterval,
	// wait CheckInterval + Jitter to get past it
	time.Sleep(prePull.CheckInterval + prePull.Jitter)

	// Now we're past the CheckInterval, NeedsPrePull() should be again true
	require.True(t, prePull.NeedsPrePull())
}
