package boolevator

import (
	"context"
	"regexp"
	"strconv"
	"strings"
)

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

func opIn(a, b interface{}) (interface{}, error) {
	if err := handleError(a, b); err != nil {
		return nil, err
	}

	return strconv.FormatBool(strings.Contains(b.(string), a.(string))), nil
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

func opEquals(a, b interface{}) (interface{}, error) {
	if err := handleError(a, b); err != nil {
		return nil, err
	}

	return strconv.FormatBool(a == b), nil
}

func opNotEquals(a, b interface{}) (interface{}, error) {
	if err := handleError(a, b); err != nil {
		return nil, err
	}

	return strconv.FormatBool(a != b), nil
}

func opRegexEquals(a, b interface{}) (interface{}, error) {
	if err := handleError(a, b); err != nil {
		return nil, err
	}

	// We don't check for the errors here because we don't know
	// which operand (left or right) is the actual regular expression
	equalsOneWay, _ := regexp.MatchString(PrepareRegexp(a.(string)), b.(string))
	equalsOtherWay, _ := regexp.MatchString(PrepareRegexp(b.(string)), a.(string))

	return strconv.FormatBool(equalsOneWay || equalsOtherWay), nil
}

func opRegexNotEquals(a, b interface{}) (interface{}, error) {
	if err := handleError(a, b); err != nil {
		return nil, err
	}

	result, err := opRegexEquals(a, b)
	if err != nil {
		return "", nil
	}

	return strconv.FormatBool(result == "false"), nil
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

// PrepareRegexp ensures that:
//
// * we match the whole string to avoid partial match false positives (e.g. we don't want ".*smh.*" regular expression
//   to match an empty string "")
// * enable Pattern.DOTALL[1] alternative in Go because otherwise simply adding ^ and $ will be too restricting,
//   since we actually support multi-line matches
// * enable Pattern.CASE_INSENSITIVE alternative in Go to be compatible with the Cirrus Cloud parser
//
// [1]: https://docs.oracle.com/javase/7/docs/api/java/util/regex/Pattern.html#DOTALL
// [2]: https://docs.oracle.com/javase/7/docs/api/java/util/regex/Pattern.html#CASE_INSENSITIVE
func PrepareRegexp(r string) string {
	return "(?s)(?i)^" + r + "$"
}
