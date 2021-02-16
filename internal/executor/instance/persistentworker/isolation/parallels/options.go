package parallels

import "github.com/cirruslabs/cirrus-cli/internal/logger"

type Option func(parallels *Parallels)

func WithLogger(logger logger.Lightweight) Option {
	return func(parallels *Parallels) {
		parallels.logger = logger
	}
}
