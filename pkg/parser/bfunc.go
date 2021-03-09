package parser

import (
	"errors"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/go-java-glob"
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
			pattern, ok := argument.(string)
			if !ok {
				return ErrBfuncArgumentIsNotString
			}

			re, err := glob.ToRegexPattern(pattern, false)
			if err != nil {
				return err
			}

			for _, affectedFile := range p.affectedFiles {
				if re.MatchString(affectedFile) {
					return "true"
				}
			}
		}

		return "false"
	}
}
