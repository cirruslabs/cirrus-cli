package remoteagent_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/remoteagent"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTimeSyncCommand(t *testing.T) {
	// https://www.epochconverter.com/?q=1234567890
	timeFixture := time.Unix(1234567890, 0).UTC()

	assert.Equal(t, "sudo -n date -u 021323312009.30\n", remoteagent.TimeSyncCommand(timeFixture))
}
