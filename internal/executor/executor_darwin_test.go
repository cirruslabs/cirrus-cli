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

func TestExecutorTart(t *testing.T) {
	vm, vmOk := os.LookupEnv("CIRRUS_INTERNAL_TART_VM")
	user, userOk := os.LookupEnv("CIRRUS_INTERNAL_TART_SSH_USER")
	password, passwordOk := os.LookupEnv("CIRRUS_INTERNAL_TART_SSH_PASSWORD")
	if !vmOk || !userOk || !passwordOk {
		t.SkipNow()
	}

	configs := map[string]string{
		"persistent_worker": fmt.Sprintf(`persistent_worker:
  isolation:
    tart:
      image: %s
      user: %s
      password: %s

task:
  tart_check_script: true
  ls_script: ls
`, vm, user, password),
		"macos_instance": fmt.Sprintf(`macos_instance:
  image: %s
  user: %s
  password: %s

task:
  tart_check_script: true
  ls_script: ls
`, vm, user, password),
	}

	for name, config := range configs {
		t.Run(name, func(t *testing.T) {
			if err := ioutil.WriteFile(".cirrus.yml", []byte(config), 0600); err != nil {
				t.Fatal(err)
			}
			if err := ioutil.WriteFile("foo.txt", []byte(config), 0600); err != nil {
				t.Fatal(err)
			}

			// Create os.Stderr writer that duplicates it's output to buf
			buf := bytes.NewBufferString("")
			writer := io.MultiWriter(os.Stderr, buf)

			renderer := renderers.NewSimpleRenderer(writer, nil)
			logger := echelon.NewLogger(echelon.TraceLevel, renderer)

			err := testutil.ExecuteWithOptions(t, ".", executor.WithLogger(logger))
			assert.NoError(t, err)

			assert.Contains(t, buf.String(), "'tart_check' script succeeded")
			assert.Contains(t, buf.String(), "'ls' script succeeded")

			assert.NotContains(t, buf.String(), "foo.txt")

			// Ensure we get the logs from the VM
			assert.Contains(t, buf.String(), "Getting initial commands...")
			assert.Contains(t, buf.String(), "Sending heartbeat...")
			assert.Contains(t, buf.String(), "Background commands to clean up after:")
		})
	}
}

func TestTartMountedWorkingDirectory(t *testing.T) {
	vm, vmOk := os.LookupEnv("CIRRUS_INTERNAL_TART_VM")
	user, userOk := os.LookupEnv("CIRRUS_INTERNAL_TART_SSH_USER")
	password, passwordOk := os.LookupEnv("CIRRUS_INTERNAL_TART_SSH_PASSWORD")
	if !vmOk || !userOk || !passwordOk {
		t.SkipNow()
	}

	config := fmt.Sprintf(`macos_instance:
  image: %s
  user: %s
  password: %s

task:
  ls_script: ls
`, vm, user, password)

	if err := ioutil.WriteFile(".cirrus.yml", []byte(config), 0600); err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile("foo.txt", []byte(config), 0600); err != nil {
		t.Fatal(err)
	}

	// Create os.Stderr writer that duplicates it's output to buf
	buf := bytes.NewBufferString("")
	writer := io.MultiWriter(os.Stderr, buf)

	renderer := renderers.NewSimpleRenderer(writer, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	err := testutil.ExecuteWithOptions(t, ".", executor.WithLogger(logger))
	assert.NoError(t, err)

	assert.Contains(t, buf.String(), "'ls' script succeeded")
	assert.Contains(t, buf.String(), "foo.txt")
}
