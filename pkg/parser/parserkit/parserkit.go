package parserkit

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/issue"
)

type ParserKit struct {
	Boolevator *boolevator.Boolevator
	IssueRegistry *issue.IssueRegistry
}
