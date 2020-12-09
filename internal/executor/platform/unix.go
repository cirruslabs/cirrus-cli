package platform

import (
	"fmt"
	"path"
)

type UnixPlatform struct{}

func NewUnix() Platform {
	return &UnixPlatform{}
}

func (platform *UnixPlatform) ProjectDirMountpoint() string {
	return "/project-dir"
}

func (platform *UnixPlatform) WorkingVolumeMountpoint() string {
	return "/tmp/cirrus-ci"
}

func (platform *UnixPlatform) AgentImage(version string) string {
	return AgentImageBase + version
}

func (platform *UnixPlatform) CopyCommand(populate bool) []string {
	copyCmd := fmt.Sprintf("cp /bin/cirrus-ci-agent %s",
		path.Join(platform.WorkingVolumeMountpoint(), WorkingVolumeAgentBinary))

	if populate {
		copyCmd += fmt.Sprintf(" && rsync -r --filter=':- .gitignore' %s/ %s",
			platform.ProjectDirMountpoint(),
			path.Join(platform.WorkingVolumeMountpoint(), WorkingVolumeWorkingDir))
	}

	return []string{"/bin/sh", "-c", copyCmd}
}

func (platform *UnixPlatform) AgentBinaryPath() string {
	return path.Join(platform.WorkingVolumeMountpoint(), WorkingVolumeAgentBinary)
}
