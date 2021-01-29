package parallels_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/parallels"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestTimeSyncCommand(t *testing.T) {
	// https://www.epochconverter.com/?q=1234567890
	timeFixture := time.Unix(1234567890, 0).UTC()

	assert.Equal(t, "sudo date 021323312009\n", parallels.TimeSyncCommand(timeFixture))
}
