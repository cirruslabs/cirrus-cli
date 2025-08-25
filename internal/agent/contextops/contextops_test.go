package contextops_test

import (
	"context"
	"testing"
	"time"

	"github.com/cirruslabs/cirrus-cli/internal/agent/contextops"
	"github.com/stretchr/testify/require"
)

func TestAndLeftToRight(t *testing.T) {
	firstCtx, firstCtxCancel := context.WithCancel(context.Background())
	firstCtxCancel()

	secondCtx, secondCtxCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer secondCtxCancel()

	waitStart := time.Now()
	<-contextops.And(firstCtx, secondCtx).Done()
	waitStop := time.Now()

	require.GreaterOrEqual(t, waitStop.Sub(waitStart), 15*time.Second)
}

func TestAndRightToLeft(t *testing.T) {
	firstCtx, firstCtxCancel := context.WithCancel(context.Background())
	firstCtxCancel()

	secondCtx, secondCtxCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer secondCtxCancel()

	waitStart := time.Now()
	<-contextops.And(secondCtx, firstCtx).Done()
	waitStop := time.Now()

	require.GreaterOrEqual(t, waitStop.Sub(waitStart), 15*time.Second)
}
