package cirrusenv_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/cirrusenv"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"testing"
)

func TestCirrusEnvConcurrentAccess(t *testing.T) {
	ce, err := cirrusenv.New("42")
	require.NoError(t, err)
	defer ce.Close()

	cmd := exec.Command("cmd.exe", "/c", "echo A=B> "+ce.Path())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())

	env, err := ce.Consume()
	require.NoError(t, err)
	require.Equal(t, map[string]string{"A": "B"}, env)
}
