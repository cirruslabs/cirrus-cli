package boolevator

import (
	"context"
	"errors"
	"fmt"
	"github.com/PaesslerAG/gval"
	"strconv"
	"strings"
	"text/scanner"
)

type Function func(arguments ...interface{}) interface{}

var ErrInternal = errors.New("internal boolevator error")

func parseString(ctx context.Context, parser *gval.Parser) (gval.Evaluable, error) {
	// Work around text/scanner stopping at newline when scanning strings
	unescaped := strings.ReplaceAll(parser.TokenText(), "\\n", "\n")

	unquoted := strings.Trim(unescaped, "'\"")

	return parser.Const(unquoted), nil
}

func Eval(expr string, env map[string]string, functions map[string]Function) (bool, error) {
	// Work around text/scanner stopping at newline when scanning strings
	expr = strings.ReplaceAll(expr, "\n", "\\n")

	// We declare this as a closure since we need a way to pass the env map inside
	expandVariable := func(ctx context.Context, parser *gval.Parser) (gval.Evaluable, error) {
		var variableName string

		r := parser.Scan()
		if r == '{' {
			/* ${VARIABLE} */
			parser.Scan()
			variableName = parser.TokenText()
			parser.Scan()
		} else {
			/* $VARIABLE */
			variableName = parser.TokenText()
		}

		// Lookup variable
		expandedVariable := env[variableName]

		return parser.Const(expandedVariable), nil
	}

	languageBases := []gval.Language{
		// Constants
		gval.Constant("true", "true"),
		gval.Constant("false", "false"),
		// gval-provided prefixes and meta prefixes
		gval.Parentheses(),
		gval.Ident(),
		// Prefixes
		gval.PrefixExtension(scanner.Char, parseString),
		gval.PrefixExtension(scanner.String, parseString),
		gval.PrefixExtension('$', expandVariable),
		// Operators
		gval.PrefixOperator("!", opNot),
		gval.InfixOperator("in", opIn),
		gval.InfixOperator("&&", opAnd),
		gval.InfixOperator("||", opOr),
		gval.InfixOperator("==", opEquals),
		gval.InfixOperator("!=", opNotEquals),
		gval.InfixOperator("=~", opRegexEquals),
		gval.InfixOperator("!=~", opRegexNotEquals),
		// Operator precedence
		//
		// Identical to https://introcs.cs.princeton.edu/java/11precedence/
		// except for the "in" and regex operators which have the same precedence
		// as their non-regex counterparts.
		gval.Precedence("!", 14),
		gval.Precedence("in", 10),
		gval.Precedence("==", 8),
		gval.Precedence("!=", 8),
		gval.Precedence("=~", 8),
		gval.Precedence("!=~", 8),
		gval.Precedence("&&", 4),
		gval.Precedence("||", 3),
	}

	// Functions
	for name, function := range functions {
		languageBases = append(languageBases, gval.Function(name, function))
	}

	result, err := gval.NewLanguage(languageBases...).Evaluate(expr, nil)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrInternal, err)
	}

	booleanValue, err := strconv.ParseBool(result.(string))
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrInternal, err)
	}

	return booleanValue, nil
}
