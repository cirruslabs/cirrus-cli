package rpc

import (
	"github.com/cirruslabs/echelon"
)

type Option func(*RPC)

func WithLogger(logger *echelon.Logger) Option {
	return func(r *RPC) {
		r.logger = logger
	}
}

func WithAdditionalEndpoint(ip string) Option {
	return func(r *RPC) {
		r.additionalEndpointIP = ip
	}
}
