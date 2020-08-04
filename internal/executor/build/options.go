package build

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/build/filter"
)

type Option func(*Build)

func WithTaskFilter(taskFilter filter.TaskFilter) Option {
	return func(b *Build) {
		b.taskFilter = taskFilter
	}
}
