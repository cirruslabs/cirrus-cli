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

var stubLogger = echelon.NewLogger(echelon.ErrorLevel, &renderers.StubRenderer{})

type RunConfig struct {
	ContainerBackendType       string
	ProjectDir                 string
	Endpoint                   endpoint.Endpoint
	ServerSecret, ClientSecret string
	TaskID                     int64
	logger                     *echelon.Logger
	DirtyMode                  bool
	ContainerOptions           options.ContainerOptions
	TartOptions                options.TartOptions
	VetuOptions                options.VetuOptions
	agentVersion               string
	containerBackend           containerbackend.ContainerBackend
	AdditionalEnvironment      map[string]string
}

func (rc *RunConfig) GetContainerBackend() (containerbackend.ContainerBackend, error) {
	if rc.containerBackend != nil {
		return rc.containerBackend, nil
	}

	if rc.ContainerBackendType == "" {
		rc.ContainerBackendType = containerbackend.BackendTypeAuto
	}

	backend, err := containerbackend.New(rc.ContainerBackendType)
	if err != nil {
		return nil, err
	}
	rc.containerBackend = backend

	return rc.containerBackend, nil
}

func (rc *RunConfig) Logger() *echelon.Logger {
	if rc.logger == nil {
		return stubLogger
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

func (rc *RunConfig) SetCLIVersion(agentVersion string) {
	rc.agentVersion = agentVersion
}

func (rc *RunConfig) SetCLIVersionWithoutDowngrade(agentVersion string) error {
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
