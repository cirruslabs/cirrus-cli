package parser

import (
	"errors"
	"github.com/bmatcuk/doublestar"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
)

var (
	ErrBfuncNoArguments         = errors.New("no arguments provided")
	ErrBfuncArgumentIsNotString = errors.New("argument should be a string")
)

func (p *Parser) bfuncChangesInclude() boolevator.Function {
	return func(arguments ...interface{}) interface{} {
		if len(arguments) == 0 {
			return ErrBfuncNoArguments
		}

		for _, argument := range arguments {
			for _, affectedFile := range p.affectedFiles {
				pattern, ok := argument.(string)
				if !ok {
					return ErrBfuncArgumentIsNotString
				}

				matched, err := doublestar.Match(pattern, affectedFile)
				if err != nil {
					return err
				}

				if matched {
					return "true"
				}
			}
		}

		return "false"
	}
}
