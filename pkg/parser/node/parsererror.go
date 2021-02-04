package node

import "github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"

func (node *Node) ParserError(format string, args ...interface{}) error {
	return parsererror.NewRich(node.Line, node.Column, format, args...)
}
