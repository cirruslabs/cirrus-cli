package tart

import (
	"github.com/cirruslabs/cirrus-cli/internal/logger"
)

type Option func(*Tart)

func WithLogger(logger logger.Lightweight) Option {
	return func(tart *Tart) {
		tart.logger = logger
	}
}

func WithSoftnet() Option {
	return func(tart *Tart) {
		tart.softnet = true
	}
}
