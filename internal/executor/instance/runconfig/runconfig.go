package runconfig

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/endpoint"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/containerbackend"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
	"github.com/hashicorp/go-version"
)

type RunConfig struct {
	ContainerBackend           containerbackend.ContainerBackend
	ProjectDir                 string
	Endpoint                   endpoint.Endpoint
	ServerSecret, ClientSecret string
	TaskID                     int64
	logger                     *echelon.Logger
	DirtyMode                  bool
	ContainerOptions           options.ContainerOptions
	TartOptions                options.TartOptions
	agentVersion               string
}

func (rc *RunConfig) Logger() *echelon.Logger {
	if rc.logger == nil {
		rc.logger = echelon.NewLogger(echelon.ErrorLevel, &renderers.StubRenderer{})
	}

	return rc.logger
}

func (rc *RunConfig) SetLogger(logger *echelon.Logger) {
	rc.logger = logger
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

func (rc *RunConfig) SetAgentVersionWithoutDowngrade(agentVersion string) error {
	if agentVersion == "" {
		return nil
	}

	requestedVersion, err := version.NewVersion(agentVersion)
	if err != nil {
		return err
	}
	defaultVersion := version.Must(version.NewVersion(platform.DefaultAgentVersion))

	if requestedVersion.LessThan(defaultVersion) {
		rc.agentVersion = defaultVersion.String()
	} else {
		rc.agentVersion = requestedVersion.String()
	}

	return nil
}
