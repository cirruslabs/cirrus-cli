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

func WithArtifactsDir(artifactsDir string) Option {
	return func(r *RPC) {
		r.artifactsDir = artifactsDir
	}
}
