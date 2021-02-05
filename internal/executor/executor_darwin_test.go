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
	"testing"
)

func TestExecutorParallels(t *testing.T) {
	testutil.TempChdir(t)

	image, imageOk := os.LookupEnv("CIRRUS_INTERNAL_PARALLELS_VM")
	user, userOk := os.LookupEnv("CIRRUS_INTERNAL_PARALLELS_SSH_USER")
	password, passwordOk := os.LookupEnv("CIRRUS_INTERNAL_PARALLELS_SSH_PASSWORD")
	if !imageOk || !userOk || !passwordOk {
		t.SkipNow()
	}

	config := fmt.Sprintf(`persistent_worker:
  isolation:
    parallels:
      image: %s
      user: %s
      password: %s
      platform: darwin

task:
  parallels_check_script: true
`, image, user, password)

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
}
