package filter

import (
	"strings"
)

type TaskFilter func(string) bool

func MatchAnyTask() TaskFilter {
	return func(taskName string) bool {
		return true
	}
}

func MatchTaskByPattern(pattern string) TaskFilter {
	patternPieces := strings.Split(pattern, ",")

	for i := range patternPieces {
		patternPieces[i] = strings.Trim(patternPieces[i], " ")
	}

	return func(taskName string) bool {
		for _, patternPiece := range patternPieces {
			if strings.EqualFold(patternPiece, taskName) {
				return true
			}
		}

		return false
	}
}
