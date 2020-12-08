package platform

import (
	"fmt"
	"path/filepath"
)

type WindowsPlatform struct{}

func NewWindows() Platform {
	return &WindowsPlatform{}
}

func (platform *WindowsPlatform) ProjectDirMountpoint() string {
	return "C:\\project-dir"
}

func (platform *WindowsPlatform) WorkingVolumeMountpoint() string {
	return "C:\\Windows\\Temp"
}

func (platform *WindowsPlatform) AgentImage(version string) string {
	return "mcr.microsoft.com/windows/servercore:ltsc2019"
}

func (platform *WindowsPlatform) CopyCommand(populate bool) []string {
	windowsAgentURL := fmt.Sprintf("https://github.com/cirruslabs/cirrus-ci-agent/releases/"+
		"download/v%s/agent-windows-amd64.exe", DefaultAgentVersion)

	copyCmd := fmt.Sprintf("(New-Object System.Net.WebClient).DownloadFile(\"%s\", \"%s\")",
		windowsAgentURL, platform.AgentBinaryPath())

	if populate {
		copyCmd += fmt.Sprintf("; echo D | xcopy /Y /E /H %s %s",
			platform.ProjectDirMountpoint(),
			filepath.Join(platform.WorkingVolumeMountpoint(), WorkingVolumeWorkingDir))
	}

	return []string{"powershell", copyCmd}
}

func (platform *WindowsPlatform) AgentBinaryPath() string {
	return filepath.Join(platform.WorkingVolumeMountpoint(), WorkingVolumeAgentBinary)
}
