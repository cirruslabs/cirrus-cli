package issue

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
)

const pathYAML = ".cirrus.yml"

type Registry struct {
	issues []*api.Issue
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (registry *Registry) RegisterIssuef(
	level api.Issue_Level,
	line int,
	column int,
	format string,
	args ...interface{},
) {
	registry.issues = append(registry.issues, &api.Issue{
		Level:   level,
		Message: fmt.Sprintf(format, args...),
		Path:    pathYAML,
		Line:    uint64(line),
		Column:  uint64(column),
	})
}

func (registry *Registry) Issues() []*api.Issue {
	return registry.issues
}
