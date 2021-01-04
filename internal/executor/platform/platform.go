package platform

const (
	// WorkingVolumeWorkingDir is a working directory relative to the WorkingVolumeMountpoint().
	WorkingVolumeWorkingDir = "working-dir"

	// WorkingVolumeAgentBinary is the name of the agent binary relative to the WorkingVolumeMountpoint().
	WorkingVolumeAgentBinary = "cirrus-ci-agent"

	// AgentImageBase is used as a prefix to the agent's version to craft the full agent image name.
	AgentImageBase = "gcr.io/cirrus-ci-community/cirrus-ci-agent:v"

	// DefaultAgentVersion represents the default version of the https://github.com/cirruslabs/cirrus-ci-agent to use.
	DefaultAgentVersion = "1.26.0"
)

type Platform interface {
	// Where we will mount the project directory for further copying into a working volume
	ProjectDirMountpoint() string

	// Where working volume is mounted to when running Docker containers on Unix-like platforms.
	WorkingVolumeMountpoint() string

	AgentImage(version string) string

	CopyCommand(populate bool) []string

	AgentBinaryPath() string
}
