package evaluator

import "net/http"

type Option func(r *ConfigurationEvaluatorServiceServer)

func WithRoundTripperForTests(roundTripperForTests http.RoundTripper) Option {
	return func(r *ConfigurationEvaluatorServiceServer) {
		r.roundTripperForTests = roundTripperForTests
	}
}
