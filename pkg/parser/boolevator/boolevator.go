package boolevator

import (
	"context"
	"errors"
	"fmt"
	"github.com/PaesslerAG/gval"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/expander"
	"strconv"
	"text/scanner"
)

type Function func(arguments ...interface{}) interface{}

type Boolevator struct {
	functions map[string]Function
}

var ErrInternal = errors.New("internal boolevator error")

func New(opts ...Option) *Boolevator {
	boolevator := &Boolevator{
		functions: make(map[string]Function),
	}

	// Apply options
	for _, opt := range opts {
		opt(boolevator)
	}

	return boolevator
}

func parseString(_ context.Context, parser *gval.Parser) (gval.Evaluable, error) {
	unquoted := trimAllQuotes(parser.TokenText())

	return parser.Const(unquoted), nil
}

func trimAllQuotes(s string) string {
	if len(s) < 2 {
		return s
	}
	firstCharacter := s[0]
	lastCharacter := s[len(s)-1]
	if firstCharacter == lastCharacter && (firstCharacter == '"' || firstCharacter == '\'') {
		// return recursively to handle double quoted strings
		return trimAllQuotes(s[1 : len(s)-1])
	}
	return s
}

func (boolevator *Boolevator) Eval(expr string, env map[string]string) (bool, error) {
	// Ensure that we keep the env as is
	localEnv := make(map[string]string)
	for key, value := range env {
		localEnv[key] = expander.ExpandEnvironmentVariables(value, env)
	}

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
		expandedVariable := localEnv[variableName]

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
	for name, function := range boolevator.functions {
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
