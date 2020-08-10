package executor

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/taskfilter"
	"github.com/sirupsen/logrus"
)

type Option func(*Executor)

func WithLogger(logger *logrus.Logger) Option {
	return func(e *Executor) {
		e.logger = logger
	}
}

func WithTaskFilter(taskFilter taskfilter.TaskFilter) Option {
	return func(e *Executor) {
		e.taskFilter = taskFilter
	}
}

func WithEnvironment(environment map[string]string) Option {
	return func(e *Executor) {
		e.userSpecifiedEnvironment = environment
	}
}
