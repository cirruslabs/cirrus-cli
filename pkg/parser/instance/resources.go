package instance

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"strconv"
	"strings"
	"unicode"
)

const (
	usabilityMebibyteBorder = 100
	kibi                    = 1024
)

func ParseMegaBytes(s string) (uint32, error) {
	// Split the string into two parts
	sLower := strings.ToLower(s)
	cutAfter := strings.LastIndexFunc(sLower, unicode.IsDigit)
	digitsPart := sLower[:cutAfter+1]
	suffixPart := sLower[cutAfter+1:]

	// Parse the digits part
	memory, err := strconv.ParseUint(digitsPart, 10, 32)
	if err != nil {
		return 0, err
	}
	memoryResult := uint32(memory)

	// Modify the digits part depending on the suffix part
	switch suffixPart {
	case "":
		// Usability: values less than usabilityMebibyteBorder as are treated as gibibytes
		if memoryResult < usabilityMebibyteBorder {
			memoryResult *= kibi
		}
	case "mb":
		fallthrough
	case "mi":
		fallthrough
	case "m":
		break
	case "gb":
		fallthrough
	case "gi":
		fallthrough
	case "g":
		memoryResult *= kibi
	default:
		return 0, fmt.Errorf("%w: unsupported digital information unit suffix: '%s'", parsererror.ErrParsing, suffixPart)
	}

	return memoryResult, nil
}
