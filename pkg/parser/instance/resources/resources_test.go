package resources_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/instance/resources"
	"github.com/stretchr/testify/assert"
	"testing"
)

func parseMegaBytesHelper(t *testing.T, s string) int64 {
	result, err := resources.ParseMegaBytes(s)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestParseMegaBytes(t *testing.T) {
	assert.EqualValues(t, 8*1024, parseMegaBytesHelper(t, "8"))
	assert.EqualValues(t, 128, parseMegaBytesHelper(t, "128"))
	assert.EqualValues(t, 128, parseMegaBytesHelper(t, "128MB"))
	assert.EqualValues(t, 5*1024, parseMegaBytesHelper(t, "5G"))
}
