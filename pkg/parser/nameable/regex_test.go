package nameable_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRegexNameable(t *testing.T) {
	assert.True(t, nameable.NewRegexNameable("").Matches(""))
	assert.True(t, nameable.NewRegexNameable("a").Matches("a"))
	assert.False(t, nameable.NewRegexNameable("a").Matches("b"))

	assert.True(t, nameable.NewRegexNameable("^(.*)b$").Matches("ab"))
	assert.False(t, nameable.NewRegexNameable("^(.*)c$").Matches("ab"))
}

func TestFirstGroupOrDefault(t *testing.T) {
	const defaultValue = "main"

	name := nameable.NewRegexNameable("(.*)task")
	assert.Equal(t, defaultValue, name.FirstGroupOrDefault("123", defaultValue))
	assert.Equal(t, defaultValue, name.FirstGroupOrDefault("task", defaultValue))
	assert.Equal(t, "a", name.FirstGroupOrDefault("a_task", defaultValue))
}
