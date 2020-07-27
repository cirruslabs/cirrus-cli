package executor

import "github.com/sirupsen/logrus"

type Option func(*Executor)

func WithLogger(logger *logrus.Logger) Option {
	return func(e *Executor) {
		e.logger = logger
	}
}
