package boolevator_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

// TestEnsureFullMultilineMatch ensures that a regular expression passed to PrepareRegexp()
// will be forced to match the whole string instead of only a part of it.
func TestEnsureFullMultilineMatch(t *testing.T) {
	match, err := regexp.MatchString(boolevator.PrepareRegexp("s"), "something")
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, match)
}
