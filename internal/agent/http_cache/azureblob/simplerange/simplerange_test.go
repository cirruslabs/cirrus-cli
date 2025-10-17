package simplerange_test

import (
	"testing"

	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/azureblob/simplerange"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	start, end, err := simplerange.Parse("bytes=10-")
	require.NoError(t, err)
	require.EqualValues(t, 10, start)
	require.Nil(t, end)

	start, end, err = simplerange.Parse("bytes=10-50")
	require.NoError(t, err)
	require.EqualValues(t, 10, start)
	require.NotNil(t, end)
	require.EqualValues(t, 50, *end)

	_, _, err = simplerange.Parse("bytes=10-50, 100-150")
	require.Error(t, err)

	_, _, err = simplerange.Parse("bytes=-10")
	require.Error(t, err)
}
