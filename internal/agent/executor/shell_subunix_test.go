//go:build unix && !(openbsd || netbsd)

package executor

import (
	"context"
	"github.com/mitchellh/go-ps"
	"github.com/stretchr/testify/assert"
	"regexp"
	"strconv"
	"testing"
	"time"
)

// TestProcessGroupTermination ensures that we terminate the whole process group that
// the shell spawned in ShellCommandsAndGetOutput() has been placed into, thus killing
// it's children processes.
func TestProcessGroupTermination(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	success, output := ShellCommandsAndGetOutput(ctx, []string{"sleep 86400 & echo target PID is $! ; sleep 60"}, nil)

	assert.False(t, success, "the command should fail due to time out error")
	assert.Contains(t, output, "Timed out!", "the command should time out")

	re := regexp.MustCompile(".*target PID is ([0-9]+).*")
	matches := re.FindStringSubmatch(output)
	if len(matches) != 2 {
		t.Fatal("failed to find target PID")
	}

	pid, err := strconv.ParseInt(matches[1], 10, 32)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the zombie to be reaped by the init process
	time.Sleep(5 * time.Second)

	// Unfortunately go-ps error behavior differs depending on the OS,
	// (missing process is an error on FreeBSD but there's no error on Linux),
	// so we skip the check here
	process, _ := ps.FindProcess(int(pid))
	assert.Nil(t, process)
}
