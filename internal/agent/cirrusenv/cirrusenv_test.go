package cirrusenv_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/cirrusenv"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestCirrusEnvNormal(t *testing.T) {
	ce, err := cirrusenv.New("42")
	if err != nil {
		t.Fatal(err)
	}
	defer ce.Close()

	if err := os.WriteFile(ce.Path(), []byte("A=B\nA==B"), 0600); err != nil {
		t.Fatal(err)
	}

	env, err := ce.Consume()
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]string{
		"A": "=B",
	}

	assert.Equal(t, expected, env)
}
