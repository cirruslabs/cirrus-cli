package boolevator

import (
	"context"
	"regexp"
	"strconv"
	"strings"
)

func opNot(ctx context.Context, parameter interface{}) (interface{}, error) {
	val, err := strconv.ParseBool(parameter.(string))
	if err != nil {
		return "", err
	}

	return strconv.FormatBool(!val), nil
}

func opIn(a, b interface{}) (interface{}, error) {
	return strconv.FormatBool(strings.Contains(b.(string), a.(string))), nil
}

func opAnd(a, b interface{}) (interface{}, error) {
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
	return strconv.FormatBool(a == b), nil
}

func opNotEquals(a, b interface{}) (interface{}, error) {
	return strconv.FormatBool(a != b), nil
}

func opRegexEquals(a, b interface{}) (interface{}, error) {
	equalsOneWay, err := regexp.MatchString(a.(string), b.(string))
	if err != nil {
		return false, err
	}

	equalsOtherWay, err := regexp.MatchString(b.(string), a.(string))
	if err != nil {
		return false, err
	}

	return strconv.FormatBool(equalsOneWay || equalsOtherWay), nil
}

func opRegexNotEquals(a, b interface{}) (interface{}, error) {
	result, err := opRegexEquals(a, b)
	if err != nil {
		return "", nil
	}

	return strconv.FormatBool(result == "false"), nil
}
