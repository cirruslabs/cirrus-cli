package fallbackroundtripper

import (
	"github.com/deckarep/golang-set/v2"
	"log"
	"net/http"
)

var allowedStatusCodes = mapset.NewSet[int](
	http.StatusOK,
	http.StatusTemporaryRedirect,
)

type FallbackTransport struct {
	primaryTransport   http.RoundTripper
	secondaryTransport http.RoundTripper
}

func New(primaryTransport http.RoundTripper, secondaryTransport http.RoundTripper) *FallbackTransport {
	return &FallbackTransport{
		primaryTransport:   primaryTransport,
		secondaryTransport: secondaryTransport,
	}
}

func (transport *FallbackTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	// Try primary transport first
	resp, err := transport.primaryTransport.RoundTrip(request)
	if err == nil && allowedStatusCodes.ContainsOne(resp.StatusCode) {
		return resp, nil
	}

	// Fallback to secondary transport
	if err != nil {
		log.Printf("Falling back to secondary transport as primary transport failed: %v", err)
	} else {
		log.Printf("Falling back to secondary transport as primary transport failed: HTTP %d", resp.StatusCode)
	}

	return transport.secondaryTransport.RoundTrip(request)
}
