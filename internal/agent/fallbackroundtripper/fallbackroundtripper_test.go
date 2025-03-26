package fallbackroundtripper_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/fallbackroundtripper"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

type primary struct{}

func (primary *primary) RoundTrip(request *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusBadGateway}, nil
}

type secondary struct{}

func (secondary *secondary) RoundTrip(request *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK}, nil
}

func TestFallbackRoundTripper(t *testing.T) {
	roundTripper := fallbackroundtripper.New(&primary{}, &secondary{})

	req, err := http.NewRequest("GET", "http://example.com/", nil)
	require.NoError(t, err)

	resp, err := roundTripper.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
