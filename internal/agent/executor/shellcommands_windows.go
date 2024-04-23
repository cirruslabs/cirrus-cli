package executor

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/environment"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/piper"
	"golang.org/x/sys/windows"
	"os/exec"
	"strconv"
	"strings"
)

type ShellCommands struct {
	cmd            *exec.Cmd
	piper          *piper.Piper
	jobHandle      windows.Handle
	savedErrorMode *uint32
}

var ErrInvalidWindowsErrorMode = errors.New("invalid CIRRUS_WINDOWS_ERROR_MODE value")

func (sc *ShellCommands) beforeStart(env *environment.Environment) error {
	errorModeRaw, ok := env.Lookup("CIRRUS_WINDOWS_ERROR_MODE")
	if !ok {
		return nil
	}

	if !strings.HasPrefix(errorModeRaw, "0x") {
		return fmt.Errorf("%w: should start with 0x", ErrInvalidWindowsErrorMode)
	}
	errorModeRaw = strings.TrimPrefix(errorModeRaw, "0x")

	errorMode, err := strconv.ParseUint(errorModeRaw, 16, 32)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidWindowsErrorMode, err)
	}

	// Set the error mode for the child process to use
	// to work around Golang's disableWER() function[1]
	// [1]: https://github.com/golang/go/issues/9121
	savedErrorMode := windows.SetErrorMode(uint32(errorMode))
	sc.savedErrorMode = &savedErrorMode

	return nil
}

func (sc *ShellCommands) afterStart() {
	// Restore the original error mode
	if sc.savedErrorMode != nil {
		windows.SetErrorMode(*sc.savedErrorMode)
	}

	jobHandle, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return
	}
	sc.jobHandle = jobHandle

	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE,
		false, uint32(sc.cmd.Process.Pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(process)

	if err := windows.AssignProcessToJobObject(jobHandle, process); err != nil {
		return
	}
}

func (sc *ShellCommands) kill() error {
	if sc.jobHandle == 0 {
		return sc.cmd.Process.Kill()
	}

	if err := windows.TerminateJobObject(sc.jobHandle, 0); err != nil {
		return err
	}

	return windows.CloseHandle(sc.jobHandle)
}
