package executor

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/cirruslabs/cirrus-cli/internal/executor/taskfilter"
	"github.com/cirruslabs/echelon"
	"time"
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

func WithBaseEnvironmentOverride(environment map[string]string) Option {
	return func(e *Executor) {
		e.baseEnvironment = environment
	}
}

func WithUserSpecifiedEnvironment(environment map[string]string) Option {
	return func(e *Executor) {
		e.userSpecifiedEnvironment = environment
	}
}

func WithDirtyMode() Option {
	return func(e *Executor) {
		e.dirtyMode = true
	}
}

func WithHeartbeatTimeout(heartbeatTimeout time.Duration) Option {
	return func(e *Executor) {
		e.heartbeatTimeout = heartbeatTimeout
	}
}

func WithContainerOptions(containerOptions options.ContainerOptions) Option {
	return func(e *Executor) {
		e.containerOptions = containerOptions
	}
}

func WithContainerBackendType(containerBackendType string) Option {
	return func(e *Executor) {
		e.containerBackendType = containerBackendType
	}
}

func WithTartOptions(tartOptions options.TartOptions) Option {
	return func(e *Executor) {
		e.tartOptions = tartOptions
	}
}

func WithArtifactsDir(artifactsDir string) Option {
	return func(e *Executor) {
		e.artifactsDir = artifactsDir
	}
}
