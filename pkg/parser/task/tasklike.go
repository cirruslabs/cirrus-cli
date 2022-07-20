package task

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
)

type ParseableTaskLike interface {
	Name() string
	SetName(name string)
	FallbackName() string
	SetFallbackName(name string)
	Alias() string
	DependsOnNames() []string

	ID() int64
	SetID(id int64)
	SetIndexWithinBuild(id int64)

	DependsOnIDs() []int64
	SetDependsOnIDs(ids []int64)

	OnlyIfExpression() string
	Enabled(env map[string]string, boolevator *boolevator.Boolevator) (bool, error)

	Line() int
	Column() int

	parseable.Parseable
}
