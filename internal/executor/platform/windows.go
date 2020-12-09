package platform

import (
	"fmt"
	"path/filepath"
)

type WindowsPlatform struct {
	image string
}

func NewWindows(osVersion string) Platform {
	var image string

	switch osVersion {
	case "1709":
		image = "mcr.microsoft.com/windows/servercore:1709"
	case "1803":
		image = "mcr.microsoft.com/windows/servercore:1803"
	default:
		image = "mcr.microsoft.com/windows/servercore:ltsc2019"
	}

	return &WindowsPlatform{
		image: image,
	}
}

func (platform *WindowsPlatform) ProjectDirMountpoint() string {
	return "C:\\project-dir"
}

func (platform *WindowsPlatform) WorkingVolumeMountpoint() string {
	return "C:\\Windows\\Temp"
}

func (platform *WindowsPlatform) AgentImage(version string) string {
	return platform.image
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
