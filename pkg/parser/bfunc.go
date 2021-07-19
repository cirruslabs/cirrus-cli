package parser

import (
	"errors"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/go-java-glob"
	"regexp"
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

		matchedFiles, err := p.countMatchingAffectedFiles(arguments)
		if err != nil {
			return err
		}
		if matchedFiles > 0 {
			return "true"
		}
		return "false"
	}
}

func (p *Parser) bfuncChangesIncludeOnly() boolevator.Function {
	return func(arguments ...interface{}) interface{} {
		if len(arguments) == 0 {
			return ErrBfuncNoArguments
		}

		matchedFiles, err := p.countMatchingAffectedFiles(arguments)
		if err != nil {
			return err
		}
		if matchedFiles > 0 && matchedFiles == len(p.affectedFiles) {
			return "true"
		}
		return "false"
	}
}

func (p *Parser) countMatchingAffectedFiles(patters []interface{}) (count int, err error) {
	for _, pattern := range patters {
		patternExpression, ok := pattern.(string)
		if !ok {
			err = ErrBfuncArgumentIsNotString
			return
		}

		var re *regexp.Regexp
		re, err = glob.ToRegexPattern(patternExpression, false)
		if err != nil {
			return
		}

		for _, affectedFile := range p.affectedFiles {
			if re.MatchString(affectedFile) {
				count++
			}
		}
	}
	return
}
