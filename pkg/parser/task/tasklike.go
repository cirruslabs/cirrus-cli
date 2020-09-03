package task

import "github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"

type ParseableTaskLike interface {
	Name() string
	SetName(name string)
	DependsOnNames() []string

	ID() int64
	SetID(id int64)

	DependsOnIDs() []int64
	SetDependsOnIDs(ids []int64)

	parseable.Parseable
}
