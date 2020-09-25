package task_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/task"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func parseSecondsHelper(t *testing.T, s string) uint32 {
	result, err := task.ParseSeconds(s)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestParseSeconds(t *testing.T) {
	assert.EqualValues(t, (1 * time.Second).Seconds(), parseSecondsHelper(t, "1s"))
	assert.EqualValues(t, (60 * time.Second).Seconds(), parseSecondsHelper(t, "60s"))

	assert.EqualValues(t, (0 * time.Minute).Seconds(), parseSecondsHelper(t, "0"))
	assert.EqualValues(t, (1 * time.Minute).Seconds(), parseSecondsHelper(t, "1"))
	assert.EqualValues(t, (60 * time.Minute).Seconds(), parseSecondsHelper(t, "60"))

	assert.EqualValues(t, (1 * time.Minute).Seconds(), parseSecondsHelper(t, "1m"))
	assert.EqualValues(t, (5 * time.Minute).Seconds(), parseSecondsHelper(t, "5m"))

	assert.EqualValues(t, (1 * time.Hour).Seconds(), parseSecondsHelper(t, "1h"))
	assert.EqualValues(t, (12 * time.Hour).Seconds(), parseSecondsHelper(t, "12h"))
}
