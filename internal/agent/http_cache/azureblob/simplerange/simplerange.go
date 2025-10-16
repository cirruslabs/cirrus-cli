package simplerange

import (
	"fmt"
	"strconv"
	"strings"
)

func Parse(s string) (int64, *int64, error) {
	after, found := strings.CutPrefix(s, "bytes=")
	if !found {
		return 0, nil, fmt.Errorf("no \"bytes=\" prefix was found")
	}

	splits := strings.Split(after, "-")
	if len(splits) != 2 {
		return 0, nil, fmt.Errorf("expected \"<range-start>-\" or \"<range-start>-<range-end>\"")
	}
	if splits[0] == "" {
		return 0, nil, fmt.Errorf("\"<range-start>\" cannot be empty")
	}

	start, err := strconv.ParseInt(splits[0], 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("cannot parse \"<range-start>\": %w", err)
	}

	if splits[1] == "" {
		// Just <range-start>-
		return start, nil, nil
	}

	end, err := strconv.ParseInt(splits[1], 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("cannot parse \"<range-end>\": %w", err)
	}

	return start, &end, nil
}
