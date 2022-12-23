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

		rawPatterns, err := bfuncArgsToStrings(arguments)
		if err != nil {
			return err
		}

		matchedFiles, err := CountMatchingAffectedFiles(p.affectedFiles, rawPatterns)
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

		rawPatterns, err := bfuncArgsToStrings(arguments)
		if err != nil {
			return err
		}

		matchedFiles, err := CountMatchingAffectedFiles(p.affectedFiles, rawPatterns)
		if err != nil {
			return err
		}
		if matchedFiles > 0 && matchedFiles == len(p.affectedFiles) {
			return "true"
		}
		return "false"
	}
}

func bfuncArgsToStrings(arguments []interface{}) ([]string, error) {
	var result []string

	for _, pattern := range arguments {
		rawPattern, ok := pattern.(string)
		if !ok {
			return nil, ErrBfuncArgumentIsNotString
		}

		result = append(result, rawPattern)
	}

	return result, nil
}

func CountMatchingAffectedFiles(affectedFiles []string, patterns []string) (int, error) {
	matchedFilesMask := make([]bool, len(affectedFiles))

	for _, pattern := range patterns {
		re, err := glob.ToRegexPattern(pattern, false)
		if err != nil {
			return 0, err
		}

		for index, affectedFile := range affectedFiles {
			if re.MatchString(affectedFile) {
				matchedFilesMask[index] = true
			}
		}
	}

	var count int
	for _, match := range matchedFilesMask {
		if match {
			count++
		}
	}

	return count, nil
}
