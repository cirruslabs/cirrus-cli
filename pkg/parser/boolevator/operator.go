package boolevator

import (
	"context"
	"regexp"
	"strconv"
	"strings"
)

type OperatorFunc func(a, b interface{}) (interface{}, error)

func opNot(ctx context.Context, parameter interface{}) (interface{}, error) {
	if err := handleError(parameter); err != nil {
		return nil, err
	}

	val, err := strconv.ParseBool(parameter.(string))
	if err != nil {
		return "", err
	}

	return strconv.FormatBool(!val), nil
}

func (ephctx *ephemeralContext) opIn() OperatorFunc {
	return func(a, b interface{}) (interface{}, error) {
		if err := handleError(a, b); err != nil {
			return nil, err
		}

		expandedAndLowerA := strings.ToLower(ephctx.getLiteralValue(a))
		expandedAndLowerB := strings.ToLower(ephctx.getLiteralValue(b))

		return strconv.FormatBool(strings.Contains(expandedAndLowerB, expandedAndLowerA)), nil
	}
}

func opAnd(a, b interface{}) (interface{}, error) {
	if err := handleError(a, b); err != nil {
		return nil, err
	}

	left, err := strconv.ParseBool(a.(string))
	if err != nil {
		return false, err
	}
	right, err := strconv.ParseBool(b.(string))
	if err != nil {
		return false, err
	}

	return strconv.FormatBool(left && right), nil
}

func opOr(a, b interface{}) (interface{}, error) {
	if err := handleError(a, b); err != nil {
		return nil, err
	}

	left, err := strconv.ParseBool(a.(string))
	if err != nil {
		return false, err
	}
	right, err := strconv.ParseBool(b.(string))
	if err != nil {
		return false, err
	}

	return strconv.FormatBool(left || right), nil
}

func (ephctx *ephemeralContext) opEquals() OperatorFunc {
	return func(a, b interface{}) (interface{}, error) {
		if err := handleError(a, b); err != nil {
			return nil, err
		}

		expandedA := ephctx.getLiteralValue(a)
		expandedB := ephctx.getLiteralValue(b)

		return strconv.FormatBool(expandedA == expandedB), nil
	}
}

func (ephctx *ephemeralContext) opNotEquals() OperatorFunc {
	return func(a, b interface{}) (interface{}, error) {
		if err := handleError(a, b); err != nil {
			return nil, err
		}

		expandedA := ephctx.getLiteralValue(a)
		expandedB := ephctx.getLiteralValue(b)

		return strconv.FormatBool(expandedA != expandedB), nil
	}
}

func (ephctx *ephemeralContext) opRegexEquals() OperatorFunc {
	return func(a, b interface{}) (interface{}, error) {
		if err := handleError(a, b); err != nil {
			return nil, err
		}

		expandedA := ephctx.getLiteralValue(a)
		expandedB := ephctx.getLiteralValue(b)

		equalsOneWay, err := regexp.MatchString(EnsureFullMultilineMatch(expandedA), expandedB)
		if err != nil {
			return false, err
		}

		equalsOtherWay, err := regexp.MatchString(EnsureFullMultilineMatch(expandedB), expandedA)
		if err != nil {
			return false, err
		}

		return strconv.FormatBool(equalsOneWay || equalsOtherWay), nil
	}
}

func (ephctx *ephemeralContext) opRegexNotEquals() OperatorFunc {
	return func(a, b interface{}) (interface{}, error) {
		if err := handleError(a, b); err != nil {
			return nil, err
		}

		result, err := ephctx.opRegexEquals()(a, b)
		if err != nil {
			return "", nil
		}

		return strconv.FormatBool(result == "false"), nil
	}
}

// handleError is a helper to catch and propagate errors from user-defined functions
// (see boolevator.WithFunctions option for more details).
func handleError(arguments ...interface{}) error {
	for _, argument := range arguments {
		if err, ok := argument.(error); ok {
			return err
		}
	}

	return nil
}

func EnsureFullMultilineMatch(r string) string {
	var newPrefix, newSuffix string

	alreadyFullyPrefixed := strings.HasPrefix(r, "^(?s)") || strings.HasPrefix(r, "(?s)^")

	// Enable Pattern.DOTALL[1] alternative in Go because otherwise simply adding ^ and $ will be too restricting,
	// since we actually support multi-line matches.
	//
	// [1]: https://docs.oracle.com/javase/7/docs/api/java/util/regex/Pattern.html#DOTALL
	if !alreadyFullyPrefixed && !strings.HasPrefix(r, "(?s)") {
		newPrefix += "(?s)"
	}

	if !alreadyFullyPrefixed && !strings.HasPrefix(r, "^") {
		newPrefix += "^"
	}

	if !strings.HasSuffix(r, "$") {
		newSuffix += "$"
	}

	return newPrefix + r + newSuffix
}
