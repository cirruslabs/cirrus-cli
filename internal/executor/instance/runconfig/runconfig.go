package runconfig

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/cirruslabs/echelon"
)

type RunConfig struct {
	ContainerBackend           containerbackend.ContainerBackend
	ProjectDir                 string
	ContainerEndpoint          string
	DirectEndpoint             string
	ServerSecret, ClientSecret string
	TaskID                     int64
	Logger                     *echelon.Logger
	DirtyMode                  bool
	ContainerOptions           options.ContainerOptions
	agentVersion               string
}

func (rc *RunConfig) GetAgentVersion() string {
	if rc.agentVersion == "" {
		return platform.DefaultAgentVersion
	}

	return rc.agentVersion
}

func (rc *RunConfig) SetAgentVersion(agentVersion string) {
	rc.agentVersion = agentVersion
}
