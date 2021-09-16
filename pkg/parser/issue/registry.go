package issue

import (
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
)

const pathYAML = ".cirrus.yml"

type IssueRegistry struct {
	issues []*api.Issue
}

func NewRegistry() *IssueRegistry {
	return &IssueRegistry{}
}

func (ireg *IssueRegistry) RegisterIssuef(level api.Issue_Level, line int, column int, format string, args ...interface{}) {
	ireg.issues = append(ireg.issues, &api.Issue{
		Level:   level,
		Message: fmt.Sprintf(format, args...),
		Path:    pathYAML,
		Line:    uint64(line),
		Column:  uint64(column),
	})
}

func (ireg *IssueRegistry) Issues() []*api.Issue {
	return ireg.issues
}
