package none

import "github.com/cirruslabs/cirrus-cli/internal/logger"

type Option func(parallels *PersistentWorkerInstance)

func WithLogger(logger logger.Lightweight) Option {
	return func(pwi *PersistentWorkerInstance) {
		pwi.logger = logger
	}
}
