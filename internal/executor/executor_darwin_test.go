package executor_test

import (
	"bytes"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestExecutorParallels(t *testing.T) {
	testutil.TempChdir(t)

	for _, platform := range []string{"darwin", "linux"} {
		platform := platform

		t.Run(platform, func(t *testing.T) {
			commonPrefix := fmt.Sprintf("CIRRUS_INTERNAL_PARALLELS_%s_", strings.ToUpper(platform))

			imageEnvVar := commonPrefix + "VM"
			userEnvVar := commonPrefix + "SSH_USER"
			passwordEnvVar := commonPrefix + "SSH_PASSWORD"

			doSingleParallelsExecution(t, platform, imageEnvVar, userEnvVar, passwordEnvVar)
		})
	}
}

func doSingleParallelsExecution(t *testing.T, platform, imageEnvVar, userEnvVar, passwordEnvVar string) {
	image, imageOk := os.LookupEnv(imageEnvVar)
	user, userOk := os.LookupEnv(userEnvVar)
	password, passwordOk := os.LookupEnv(passwordEnvVar)
	if !imageOk || !userOk || !passwordOk {
		t.SkipNow()
	}

	config := fmt.Sprintf(`persistent_worker:
  isolation:
    parallels:
      image: %s
      user: %s
      password: %s
      platform: %s

task:
  parallels_check_script: true
`, image, user, password, platform)

	if err := ioutil.WriteFile(".cirrus.yml", []byte(config), 0600); err != nil {
		t.Fatal(err)
	}

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	renderer := renderers.NewSimpleRenderer(writer, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	err := testutil.ExecuteWithOptions(t, ".", executor.WithLogger(logger))
	assert.NoError(t, err)

	assert.Contains(t, buf.String(), "'parallels_check' script succeeded")

	// Ensure we get the logs from the VM
	assert.Contains(t, buf.String(), "Getting initial commands...")
	assert.Contains(t, buf.String(), "Sending heartbeat...")
	assert.Contains(t, buf.String(), "Background commands to clean up after:")
}
