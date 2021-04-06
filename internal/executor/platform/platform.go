package platform

const (
	// workingVolumeWorkingDir is a working directory relative to the CirrusDir().
	workingVolumeWorkingDir = "working-dir"

	// workingVolumeAgentBinary is the name of the agent binary relative to the CirrusDir().
	workingVolumeAgentBinary = "cirrus-ci-agent"

	// agentImageBase is used as a prefix to the agent's version to craft the full agent image name.
	agentImageBase = "gcr.io/cirrus-ci-community/cirrus-ci-agent:v"

	// DefaultAgentVersion represents the default version of the https://github.com/cirruslabs/cirrus-ci-agent to use.
	DefaultAgentVersion = "1.38.1"
)

type CopyCommand struct {
	Command              []string
	CopiesAgentToDir     string
	CopiesProjectFromDir string
	CopiesProjectToDir   string
}

type Platform interface {
	ContainerAgentImage(version string) string
	ContainerCopyCommand(populate bool) *CopyCommand
	ContainerAgentPath() string
	ContainerAgentVolumeDir() string

	CirrusDir() string
	GenericWorkingDir() string
}
