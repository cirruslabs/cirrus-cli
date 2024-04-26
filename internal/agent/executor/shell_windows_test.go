//go:build windows

package executor

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/environment"
	"github.com/mitchellh/go-ps"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"testing"
	"time"
)

const (
	modeProcessTreeSpawner = "process-tree-spawner"
	modeIdler              = "idler"
)

func TestMain(m *testing.M) {
	switch os.Getenv("MODE") {
	case modeProcessTreeSpawner:
		cmd := exec.Command(os.Args[0])
		cmd.Env = []string{fmt.Sprintf("MODE=%s", modeIdler)}

		if err := cmd.Start(); err != nil {
			panic(err)
		}

		fmt.Printf("target PID is %d\n", cmd.Process.Pid)

		if err := cmd.Wait(); err != nil {
			panic(err)
		}

		os.Exit(0)
	case modeIdler:
		sigChan := make(chan os.Signal)
		signal.Notify(sigChan)
		for {
			<-sigChan
		}
	}

	os.Exit(m.Run())
}

// TestProcessGroupTermination ensures that we terminate all processes we've automatically
// tainted by assigning a job object to a shell spawned in ShellCommandsAndGetOutput().
func TestJobObjectTermination(t *testing.T) {
	ctx, cancel := context.WithTimeoutCause(context.Background(), 10*time.Second, ErrTimedOut)
	defer cancel()

	success, output := ShellCommandsAndGetOutput(ctx, []string{os.Args[0]},
		environment.New(map[string]string{"MODE": modeProcessTreeSpawner}))

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

	// TerminateJobObject is asynchronous
	time.Sleep(5 * time.Second)

	process, err := ps.FindProcess(int(pid))
	assert.NoError(t, err)
	assert.Nil(t, process)
}

func Test_ShellCommands_Windows(t *testing.T) {
	test_env := environment.New(map[string]string{
		"CIRRUS_WORKING_DIR": "C:\\Windows\\TEMP",
	})
	_, output := ShellCommandsAndGetOutput(context.Background(), []string{"echo 'Foo'"}, test_env)
	expected_output := "\r\nC:\\Windows\\TEMP>call echo 'Foo' \r\n'Foo'\r\n\r\nC:\\Windows\\TEMP>if 0 NEQ 0 exit /b 0 \r\n"
	if output == expected_output {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%+q' expected '%+q'", output, expected_output)
	}
}

func Test_ShellCommands_Multiline_Windows(t *testing.T) {
	test_env := environment.New(map[string]string{
		"CIRRUS_WORKING_DIR": "C:\\Windows\\TEMP",
	})
	_, output := ShellCommandsAndGetOutput(context.Background(), []string{"echo 'Foo'", "echo 'Bar'"}, test_env)
	expected_output := "\r\nC:\\Windows\\TEMP>call echo 'Foo' \r\n'Foo'\r\n\r\nC:\\Windows\\TEMP>if 0 NEQ 0 exit /b 0 \r\n\r\nC:\\Windows\\TEMP>call echo 'Bar' \r\n'Bar'\r\n\r\nC:\\Windows\\TEMP>if 0 NEQ 0 exit /b 0 \r\n"
	if output == expected_output {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%+q' expected '%+q'", output, expected_output)
	}
}

func Test_ShellCommands_Fail_Fast_Windows(t *testing.T) {
	test_env := environment.New(map[string]string{
		"CIRRUS_WORKING_DIR": "C:\\Windows\\TEMP",
	})
	success, output := ShellCommandsAndGetOutput(context.Background(), []string{
		"echo 'Hello!'",
		"echo 'Friend!'",
		"exit 1",
		"echo 'Unreachable!'",
	}, test_env)
	if success {
		t.Error("Should fail!")
	}

	expected_output := "\r\nC:\\Windows\\TEMP>call echo 'Hello!' \r\n'Hello!'\r\n\r\nC:\\Windows\\TEMP>if 0 NEQ 0 exit /b 0 \r\n\r\nC:\\Windows\\TEMP>call echo 'Friend!' \r\n'Friend!'\r\n\r\nC:\\Windows\\TEMP>if 0 NEQ 0 exit /b 0 \r\n\r\nC:\\Windows\\TEMP>call exit 1 \r\n"
	if output == expected_output {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%+q' expected '%+q'", output, expected_output)
	}
}

func Test_ShellCommands_Environment_Windows(t *testing.T) {
	test_env := environment.New(map[string]string{
		"CIRRUS_WORKING_DIR": "C:\\Windows\\TEMP",
		"FOO":                "BAR",
	})
	_, output := ShellCommandsAndGetOutput(context.Background(), []string{
		"echo %FOO%",
	}, test_env)

	expected_output := "\r\nC:\\Windows\\TEMP>call echo BAR \r\nBAR\r\n\r\nC:\\Windows\\TEMP>if 0 NEQ 0 exit /b 0 \r\n"
	if output == expected_output {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%+q' expected '%+q'", output, expected_output)
	}
}

func Test_Exit_Code_Windows(t *testing.T) {
	test_env := environment.New(map[string]string{
		"CIRRUS_WORKING_DIR": "C:\\Windows\\TEMP",
	})
	success, output := ShellCommandsAndGetOutput(context.Background(), []string{
		"export FOO=239",
		"echo %ERRORLEVEL%",
		"echo 'Unreachable!'",
	}, test_env)

	if success {
		t.Errorf("Should've failed! '%+q'", output)
	}

	expected_output := "\r\nC:\\Windows\\TEMP>call export FOO=239 \r\n'export' is not recognized as an internal or external command,\r\noperable program or batch file.\r\n\r\nC:\\Windows\\TEMP>if 1 NEQ 0 exit /b 1 \r\n"
	if output == expected_output {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%+q' expected '%+q'", output, expected_output)
	}
}

func Test_Powershell(t *testing.T) {
	test_env := environment.New(map[string]string{
		"CIRRUS_WORKING_DIR": "C:\\Windows\\TEMP",
		"CIRRUS_SHELL":       "powershell",
	})
	success, output := ShellCommandsAndGetOutput(context.Background(), []string{
		"echo 'Foo!'",
		"echo 'Bar!'",
		"exit 1",
		"echo 'Unreachable!'",
	}, test_env)

	if success {
		t.Errorf("Should've fail! '%+q'", output)
	}

	expected_output := "Foo!\r\nBar!\r\n"
	if output == expected_output {
		t.Log("Passed")
	} else {
		t.Errorf("Wrong output: '%+q' expected '%+q'", output, expected_output)
	}
}
