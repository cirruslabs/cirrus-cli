package pathsafe

import (
	"runtime"
	"strings"
	"unicode"
)

func IsPathSafe(s string) bool {
	if s == "" {
		return false
	}

	return strings.IndexFunc(s, func(r rune) bool {
		isColon := r == ':'

		// https://stackoverflow.com/a/35352640/9316533
		if (runtime.GOOS == "windows" || runtime.GOOS == "darwin") && isColon {
			return true
		}

		return !unicode.IsLetter(r) && !unicode.IsNumber(r) &&
			r != '_' && r != '-' && r != ' ' && !isColon
	}) == -1
}
