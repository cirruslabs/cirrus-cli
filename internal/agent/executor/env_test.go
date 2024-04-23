package executor

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/environment"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func Test_DefaultValue(t *testing.T) {
	result := environment.NewEmpty().ExpandText("${TAG:latest}")
	if result == "latest" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", result)
	}
}

func Test_Simple(t *testing.T) {
	result := environment.New(map[string]string{"TAG": "foo"}).ExpandText("${TAG:latest}")
	if result == "foo" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", result)
	}
}

func Test_Simple_Windows_Style(t *testing.T) {
	result := environment.New(map[string]string{"TAG": "foo"}).ExpandText("%TAG%")
	if result == "foo" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", result)
	}
}

func Test_Environment(t *testing.T) {
	original := map[string]string{
		"GOPATH2":            "/root/go",
		"GOSRC2":             "$GOPATH2/src/github.com/some/thing",
		"CIRRUS_WORKING_DIR": "$GOSRC2",
		"SCRIPT_BASE":        "$GOSRC2/contrib/cirrus",
		"PACKER_BASE":        "${SCRIPT_BASE}/packer",
	}

	expected := map[string]string{
		"GOPATH2":            "/root/go",
		"GOSRC2":             "/root/go/src/github.com/some/thing",
		"CIRRUS_WORKING_DIR": "/root/go/src/github.com/some/thing",
		"SCRIPT_BASE":        "/root/go/src/github.com/some/thing/contrib/cirrus",
		"PACKER_BASE":        "/root/go/src/github.com/some/thing/contrib/cirrus/packer",
	}

	result := environment.ExpandEnvironmentRecursively(original)

	if reflect.DeepEqual(result, expected) {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", result)
	}
}

func Test_Recursive(t *testing.T) {
	result := environment.ExpandEnvironmentRecursively(map[string]string{"FOO": "Contains $FOO"})
	if result["FOO"] == "Contains $FOO" {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%s'", result)
	}
}

func TestEnvMapAsSlice(t *testing.T) {
	assert.Equal(t, EnvMapAsSlice(map[string]string{"A": "B"}), []string{"A=B"})
}
