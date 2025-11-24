package progressreader_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/progressreader"
	"github.com/dustin/go-humanize"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func TestProgressReader(t *testing.T) {
	// Manual invocation only
	t.Skip()

	resp, err := http.Get("http://speedtest.ams1.nl.leaseweb.net/100mb.bin")
	require.NoError(t, err)
	defer resp.Body.Close()

	progressReader := progressreader.New(resp.Body, time.Second, func(bytes int64, duration time.Duration) {
		rate := float64(bytes) / duration.Seconds()

		slog.Info("Bytes read", "bytes", bytes, "duration", duration, "rate", humanize.Bytes(uint64(rate)))
	})

	_, err = io.Copy(io.Discard, progressReader)
	require.NoError(t, err)
}
