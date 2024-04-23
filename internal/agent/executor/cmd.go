//go:build !windows

package executor

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/environment"
	"github.com/cirruslabs/cirrus-cli/internal/agent/shellwords"
	"os"
	"os/exec"
	"syscall"
)

func createCmd(scripts []string, customEnv *environment.Environment) (*exec.Cmd, *os.File, error) {
	cmdShell := "/bin/sh"
	if bashPath, err := exec.LookPath("bash"); err == nil {
		cmdShell = bashPath
	}
	if customEnv != nil {
		if customShell, ok := customEnv.Lookup("CIRRUS_SHELL"); ok {
			cmdShell = customShell
		}
	}

	if cmdShell == "direct" {
		cmdArgs := shellwords.ToArgv(customEnv.ExpandText(scripts[0]))
		if len(cmdArgs) > 1 {
			cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
			return cmd, nil, nil
		}
		cmd := exec.Command(cmdArgs[0])
		return cmd, nil, nil
	}

	scriptFile, err := TempFileName("scripts", ".sh")
	if err != nil {
		return nil, nil, err
	}
	// add shebang
	scriptFile.WriteString(fmt.Sprintf("#!%s\n", cmdShell))
	scriptFile.WriteString("set -e\n")
	scriptFile.WriteString("set -o pipefail 2>/dev/null || true\n")
	scriptFile.WriteString("set -o verbose\n")
	for i := 0; i < len(scripts); i++ {
		scriptFile.WriteString(scripts[i])
		scriptFile.WriteString("\n")
	}
	scriptFile.Close()
	scriptFile.Chmod(os.FileMode(0777))
	cmdArgs := shellwords.ToArgv(cmdShell)
	cmdArgs = append(cmdArgs, scriptFile.Name())
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)

	// Run CMD in it's own session
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	return cmd, scriptFile, nil
}
