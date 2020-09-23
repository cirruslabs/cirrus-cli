package task

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"strconv"
	"strings"
	"unicode"
)

func ParseSeconds(s string) (uint32, error) {
	// Split the string into two parts
	sLower := strings.ToLower(s)
	cutAfter := strings.LastIndexFunc(sLower, unicode.IsDigit)
	digitsPart := sLower[:cutAfter+1]
	suffixPart := sLower[cutAfter+1:]

	// Parse the digits part
	parsedDigitsPartU64, err := strconv.ParseUint(digitsPart, 10, 32)
	if err != nil {
		return 0, err
	}
	parsedDigitsPartU32 := uint32(parsedDigitsPartU64)

	// Modify the digits part depending on the suffix part
	switch suffixPart {
	case "h":
		parsedDigitsPartU32 *= 3600
	case "":
		fallthrough
	case "m":
		parsedDigitsPartU32 *= 60
	case "s":
		// nothing to do
	default:
		return 0, fmt.Errorf("%w: unsupported time unit suffix: '%s'", parsererror.ErrParsing, suffixPart)
	}

	return parsedDigitsPartU32, nil
}
