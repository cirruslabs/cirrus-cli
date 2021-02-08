package parsererror

import (
	"errors"
	"fmt"
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
