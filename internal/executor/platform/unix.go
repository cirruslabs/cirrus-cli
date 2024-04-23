package platform

import (
	"fmt"
	"path"
	"path/filepath"
)

type UnixPlatform struct{}

func NewUnix() Platform {
	return &UnixPlatform{}
}

func (platform *UnixPlatform) ContainerCLIPath() string {
	return filepath.Join(platform.ContainerAgentVolumeDir(), workingVolumeAgentBinary)
}

func (platform *UnixPlatform) ContainerAgentVolumeDir() string {
	return platform.CirrusDir()
}

func (platform *UnixPlatform) CirrusDir() string {
	return "/tmp/cirrus-ci"
}

func (platform *UnixPlatform) ContainerAgentImage(version string) string {
	return agentImageBase + version
}

func (platform *UnixPlatform) ContainerCopyCommand(populate bool) *CopyCommand {
	copyCommand := &CopyCommand{
		CopiesAgentToDir:     "/agent-volume",
		CopiesProjectFromDir: "/project-host",
		CopiesProjectToDir:   "/project-volume",
	}

	copyCmd := fmt.Sprintf("cp /usr/local/bin/cirrus %s",
		path.Join(copyCommand.CopiesAgentToDir, workingVolumeAgentBinary))

	if populate {
		copyCmd += fmt.Sprintf(" && rsync -r --filter=':- .gitignore' %s/ %s",
			copyCommand.CopiesProjectFromDir, copyCommand.CopiesProjectToDir)
	}

	copyCommand.Command = []string{"/bin/sh", "-c", copyCmd}

	return copyCommand
}

func (platform *UnixPlatform) GenericWorkingDir() string {
	return path.Join(platform.CirrusDir(), workingVolumeWorkingDir)
}
