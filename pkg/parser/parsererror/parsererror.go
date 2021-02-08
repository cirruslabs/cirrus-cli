package parsererror

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInternal = errors.New("internal error")
	ErrBasic    = errors.New("parsing error")
)

type Rich struct {
	config  string
	message string
	line    int
	column  int
}

func NewRich(line, column int, format string, args ...interface{}) *Rich {
	return &Rich{
		message: fmt.Sprintf(format, args...),
		line:    line,
		column:  column,
	}
}

func (rich *Rich) Enrich(config string) {
	rich.config = config
}

func (rich *Rich) Error() string {
	return fmt.Sprintf("parsing error: %d:%d: %s", rich.line, rich.column, rich.message)
}

func (rich *Rich) Config() string {
	return rich.config
}

func (rich *Rich) Message() string {
	return rich.message
}

func (rich *Rich) Line() int {
	return rich.line
}

func (rich *Rich) Column() int {
	return rich.column
}

func (rich *Rich) ContextLines() string {
	var result string

	const contextLines = 5

	zeroBasedLine := rich.line - 1
	zeroBasedColumn := rich.column - 1

	for i, line := range strings.Split(rich.config, "\n") {
		// Skip lines that are out of context
		if i < (zeroBasedLine-contextLines) || i > (zeroBasedLine+contextLines) {
			continue
		}

		// Insert line-numbered config line
		lineNumber := fmt.Sprintf("%d: ", i+1)
		result += lineNumber + line + "\n"

		// Insert a column indicator if this was a line with error
		if i == zeroBasedLine {
			result += strings.Repeat(" ", len(lineNumber)+zeroBasedColumn) + "^\n"
		}
	}

	return result
}
