package commands_test

import (
	"bytes"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/commands"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestRunNoHeartbeats(t *testing.T) {
	// Support Tart isolation testing configured via environment variables
	image, vmOk := os.LookupEnv("CIRRUS_INTERNAL_TART_VM")
	user, userOk := os.LookupEnv("CIRRUS_INTERNAL_TART_SSH_USER")
	password, passwordOk := os.LookupEnv("CIRRUS_INTERNAL_TART_SSH_PASSWORD")
	if !vmOk || !userOk || !passwordOk {
		t.Skip("no Tart credentials configured")
	}

	t.Logf("Using Tart VM %s for testing...", image)

	// Craft Cirrus CI configuration
	config := fmt.Sprintf(`persistent_worker:
  isolation:
    tart:
      image: %s
      user: %s
      password: %s

task:
  # Install proctools, otherwise pgrep/pkill won't be able
  # to find the Cirrus CI Agent's process PID
  install_proctools_script:
    - brew install proctools
  script:
    - pkill -STOP cirrus-ci-agent
`, image, user, password)

	testutil.TempChdir(t)

	if err := os.WriteFile(".cirrus.yml", []byte(config), 0600); err != nil {
		t.Fatal(err)
	}

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	// Run the test
	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "--tart-lazy-pull", "-v", "-o simple"})
	command.SetOut(writer)
	command.SetErr(writer)
	err := command.Execute()

	// Ensure that we've timed out because of no heartbeats
	require.Error(t, err)
	require.Contains(t, buf.String(), "no heartbeats were received in the last ")
}

func TestRunVolumes(t *testing.T) {
	// Support Tart isolation testing configured via environment variables
	image, vmOk := os.LookupEnv("CIRRUS_INTERNAL_TART_VM")
	user, userOk := os.LookupEnv("CIRRUS_INTERNAL_TART_SSH_USER")
	password, passwordOk := os.LookupEnv("CIRRUS_INTERNAL_TART_SSH_PASSWORD")
	if !vmOk || !userOk || !passwordOk {
		t.Skip("no Tart credentials configured")
	}

	t.Logf("Using Tart VM %s for testing...", image)

	nonExistentVolumeSourcePath := filepath.Join("/tmp", uuid.NewString())
	require.True(t, pathDoesNotExist(nonExistentVolumeSourcePath),
		"non-existent volume should not exist at the beginning of the test")

	// Craft Cirrus CI configuration
	config := fmt.Sprintf(`
persistent_worker:
  isolation:
    tart:
      image: %s
      user: %s
      password: %s
      volumes:
        # Non-existent volume source should be created and cleaned-up
        - name: mounted-non-existent
          source: "%s"
        # Read-only volume without a name
        - source: "/tmp"
          target: "/Users/%s/mounted-tmp"
          readonly: true
        # Volume with a name, but without a target
        - name: mounted-var
          source: "/var"

task:
  script:
    - test -d "/Volumes/My Shared Files/mounted-non-existent"
    - test -d "/Users/%s/mounted-tmp"
    - touch "/Users/%s/mounted-tmp/test.txt" || true
    - test -d "/Volumes/My Shared Files/mounted-var"

`, image, user, password, nonExistentVolumeSourcePath, user, user, user)

	testutil.TempChdir(t)

	if err := os.WriteFile(".cirrus.yml", []byte(config), 0600); err != nil {
		t.Fatal(err)
	}

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	// Run the test
	command := commands.NewRootCmd()
	command.SetArgs([]string{"run", "--tart-lazy-pull", "-o simple"})
	command.SetOut(writer)
	command.SetErr(writer)
	err := command.Execute()

	// Ensure that we've timed out because of no heartbeats
	require.NoError(t, err)
	require.Contains(t, buf.String(), "test.txt: Operation not permitted")

	require.True(t, pathDoesNotExist(nonExistentVolumeSourcePath),
		"non-existent volume should be cleaned up at the end of the test")
}

func pathDoesNotExist(path string) bool {
	_, err := os.Stat(path)

	return os.IsNotExist(err)
}
