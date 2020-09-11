package executor

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/cirruslabs/cirrus-cli/internal/executor/taskfilter"
	"github.com/cirruslabs/echelon"
)

type Option func(*Executor)

func WithLogger(logger *echelon.Logger) Option {
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

func WithDirtyMode() Option {
	return func(e *Executor) {
		e.dirtyMode = true
	}
}

func WithDockerOptions(dockerOptions options.DockerOptions) Option {
	return func(e *Executor) {
		e.dockerOptions = dockerOptions
	}
}
