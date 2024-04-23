package executor

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/environment"
	"os"
	"os/exec"
	"strings"
)

func createCmd(scripts []string, custom_env *environment.Environment) (*exec.Cmd, *os.File, error) {
	cmdShell := "cmd.exe"
	if custom_env != nil {
		if customShell, ok := custom_env.Lookup("CIRRUS_SHELL"); ok {
			cmdShell = customShell
		}
	}

	if strings.HasSuffix(cmdShell, "powershell.exe") || strings.HasSuffix(cmdShell, "powershell") {
		return createWindowsPowershellCmd(cmdShell, scripts)
	} else if strings.HasSuffix(cmdShell, "bash.exe") || strings.HasSuffix(cmdShell, "bash") {
		return createWindowsBashCmd(cmdShell, scripts)
	} else {
		return createWindowsBatchCmd(cmdShell, scripts)
	}
}

func createWindowsBatchCmd(cmdShell string, scripts []string) (*exec.Cmd, *os.File, error) {
	scriptFile, err := TempFileName("scripts", ".bat")
	if err != nil {
		return nil, nil, err
	}
	for i := 0; i < len(scripts); i++ {
		scriptFile.WriteString("call ")
		scriptFile.WriteString(scripts[i])
		scriptFile.WriteString("\n")
		scriptFile.WriteString("if %errorlevel% neq 0 exit /b %errorlevel%\n")
	}
	scriptFile.Close()

	cmd := exec.Command(cmdShell, "/c", scriptFile.Name())
	return cmd, scriptFile, nil
}

func createWindowsBashCmd(cmdShell string, scripts []string) (*exec.Cmd, *os.File, error) {
	scriptFile, err := TempFileName("scripts", ".sh")
	if err != nil {
		return nil, nil, err
	}
	scriptFile.WriteString("set -e\n")
	if strings.Contains(cmdShell, "bash") {
		scriptFile.WriteString("set -o pipefail\n")
	}
	scriptFile.WriteString("set -o verbose\n")
	for i := 0; i < len(scripts); i++ {
		scriptFile.WriteString(scripts[i])
		scriptFile.WriteString("\n")
	}
	scriptFile.Close()

	cmd := exec.Command(cmdShell, scriptFile.Name())
	return cmd, scriptFile, nil
}

func createWindowsPowershellCmd(cmdShell string, scripts []string) (*exec.Cmd, *os.File, error) {
	scriptFile, err := TempFileName("scripts", ".ps1")
	if err != nil {
		return nil, nil, err
	}
	scriptFile.WriteString("$ErrorActionPreference = \"Stop\"\n")
	scriptFile.WriteString("$ProgressPreference = \"SilentlyContinue\"\n")
	for i := 0; i < len(scripts); i++ {
		scriptFile.WriteString(scripts[i])
		scriptFile.WriteString("\n")
	}
	scriptFile.Close()

	cmd := exec.Command(cmdShell, "-executionpolicy", "bypass", "-File", scriptFile.Name())
	return cmd, scriptFile, nil
}
