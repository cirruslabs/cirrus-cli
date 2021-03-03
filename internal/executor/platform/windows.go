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

func (platform *WindowsPlatform) ContainerAgentPath() string {
	return filepath.Join(platform.ContainerAgentVolumeDir(), workingVolumeAgentBinary)
}

func (platform *WindowsPlatform) ContainerAgentVolumeDir() string {
	return platform.CirrusDir()
}

func (platform *WindowsPlatform) CirrusDir() string {
	return "C:\\Windows\\Temp\\cirrus-ci"
}

func (platform *WindowsPlatform) ContainerAgentImage(version string) string {
	return platform.image
}

func (platform *WindowsPlatform) ContainerCopyCommand(populate bool) *CopyCommand {
	copyCommand := &CopyCommand{
		CopiesAgentToDir:     "C:\\agent-volume",
		CopiesProjectFromDir: "C:\\project-host",
		CopiesProjectToDir:   "C:\\project-volume",
	}

	windowsAgentURL := fmt.Sprintf("https://github.com/cirruslabs/cirrus-ci-agent/releases/"+
		"download/v%s/agent-windows-amd64.exe", DefaultAgentVersion)

	copyCmd := fmt.Sprintf("(New-Object System.Net.WebClient).DownloadFile(\"%s\", \"%s\")",
		windowsAgentURL, filepath.Join(copyCommand.CopiesAgentToDir, workingVolumeAgentBinary))

	if populate {
		copyCmd += fmt.Sprintf("; echo D | xcopy /Y /E /H %s %s",
			copyCommand.CopiesProjectFromDir, copyCommand.CopiesProjectToDir)
	}

	copyCommand.Command = []string{"powershell", copyCmd}

	return copyCommand
}

func (platform *WindowsPlatform) GenericWorkingDir() string {
	return filepath.Join(platform.CirrusDir(), workingVolumeWorkingDir)
}
