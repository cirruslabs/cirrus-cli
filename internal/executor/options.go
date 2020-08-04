package executor

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/build/filter"
	"github.com/sirupsen/logrus"
)

type Option func(*Executor)

func WithLogger(logger *logrus.Logger) Option {
	return func(e *Executor) {
		e.logger = logger
	}
}

func WithTaskFilter(taskFilter filter.TaskFilter) Option {
	return func(e *Executor) {
		e.taskFilter = taskFilter
	}
}
