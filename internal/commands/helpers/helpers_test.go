package helpers_test

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/commands/helpers"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestEnvFileToMap(t *testing.T) {
	tempFile, err := os.CreateTemp("", "")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tempFile.Name())
	}()

	_, err = fmt.Fprintln(tempFile, "A=B")
	require.NoError(t, err)

	t.Setenv("C", "D")
	_, err = fmt.Fprintln(tempFile, "C")
	require.NoError(t, err)

	require.NoError(t, tempFile.Close())

	env, err := helpers.EnvFileToMap(tempFile.Name())
	require.NoError(t, err)
	require.Equal(t, map[string]string{"A": "B", "C": "D"}, env)
}
