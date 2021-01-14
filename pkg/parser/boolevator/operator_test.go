package boolevator_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

// TestEnsureFullMatch ensures that a regular expression passed to EnsureFullMatch()
// will be forced to match the whole string instead of only a part of it.
func TestEnsureFullMatch(t *testing.T) {
	match, err := regexp.MatchString(boolevator.EnsureFullMatch("s"), "something")
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, match)
}
