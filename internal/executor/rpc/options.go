package rpc

import "github.com/sirupsen/logrus"

type Option func(*RPC)

func WithLogger(logger *logrus.Logger) Option {
	return func(r *RPC) {
		r.logger = logger
	}
}
