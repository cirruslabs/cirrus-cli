package executor_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestWithTimestamps(t *testing.T) {
	testCases := []struct {
		Name           string
		Input          []byte
		ExpectedOutput []byte
		OwesTimestamp  bool
	}{
		{
			"simple: text plus newline",
			[]byte("abc\n"),
			[]byte("[00:00:00.000] abc\n"),
			true,
		},
		{
			"simple: newline plus text",
			[]byte("\nabc"),
			[]byte("[00:00:00.000] \n[00:00:00.000] abc"),
			false,
		},
		{
			"simple: text",
			[]byte("abc"),
			[]byte("[00:00:00.000] abc"),
			false,
		},
		{
			"simple: newline",
			[]byte("\n"),
			[]byte("[00:00:00.000] \n"),
			true,
		},
		{
			"windows-style line endings are kept intact",
			[]byte("\r\nfirst line\r\nsecond line\r\n"),
			[]byte("[00:00:00.000] \r\n[00:00:00.000] first line\r\n[00:00:00.000] second line\r\n"),
			true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.Name, func(t *testing.T) {
			uploader := executor.LogUploader{
				LogTimestamps: true,
				GetTimestamp: func() time.Time {
					return time.Unix(0, 0).UTC()
				},
				OweTimestamp: true,
			}

			assert.Equal(t, string(testCase.ExpectedOutput), string(uploader.WithTimestamps(testCase.Input)))
			assert.Equal(t, testCase.OwesTimestamp, uploader.OweTimestamp)
		})
	}
}
