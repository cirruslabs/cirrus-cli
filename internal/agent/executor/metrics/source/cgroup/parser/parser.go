package parser

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var (
	ErrInvalidFormat = errors.New("invalid format")
	ErrInternal      = errors.New("internal parser error")
)

func ParseSingleValueFile(input io.Reader) (uint64, error) {
	scanner := bufio.NewScanner(input)

	if !scanner.Scan() {
		err := scanner.Err()
		if err == nil {
			err = io.EOF
		}

		return 0, fmt.Errorf("%w: got error while reading line: %v", ErrInvalidFormat, err)
	}

	line := scanner.Text()
	parsed, err := strconv.ParseUint(line, 10, 64)
	if err != nil {
		return 0, err
	}

	if scanner.Scan() {
		return 0, fmt.Errorf("%w: extraneous lines found", ErrInvalidFormat)
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInternal, err)
	}

	return parsed, nil
}

func ParseKeyValueFile(input io.Reader) (map[string]uint64, error) {
	result := map[string]uint64{}

	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		line := scanner.Text()

		splits := strings.Split(line, " ")
		if len(splits) != 2 {
			return nil, fmt.Errorf("%w: each line should be splittable into 2 parts, got %d parts",
				ErrInvalidFormat, len(splits))
		}

		key := splits[0]
		value := splits[1]

		parsedValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to parse value %q: %v", ErrInvalidFormat, value, err)
		}

		result[key] = parsedValue
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInternal, err)
	}

	return result, nil
}
