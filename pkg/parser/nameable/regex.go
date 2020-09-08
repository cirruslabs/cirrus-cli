package nameable

import (
	"regexp"
	"strings"
)

type RegexNameable struct {
	re *regexp.Regexp
}

func NewRegexNameable(pattern string) *RegexNameable {
	return &RegexNameable{re: regexp.MustCompile(pattern)}
}

func (rn RegexNameable) Matches(s string) bool {
	return rn.re.MatchString(s)
}

func (rn *RegexNameable) Regex() *regexp.Regexp {
	return rn.re
}

func (rn *RegexNameable) String() string {
	return rn.re.String()
}

func (rn *RegexNameable) FirstGroupOrDefault(s string, defaultValue string) string {
	submatch := rn.re.FindStringSubmatch(s)
	if submatch == nil {
		return defaultValue
	}

	if submatch[1] == "" {
		return defaultValue
	}

	return strings.TrimSuffix(submatch[1], "_")
}
