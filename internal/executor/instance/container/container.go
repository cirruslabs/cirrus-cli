package container

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/volume"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
)

type Instance struct {
	Image                string
	CPU                  float32
	Memory               uint32
	AdditionalContainers []*api.AdditionalContainer
	Architecture         *api.Architecture
	Platform             platform.Platform
	CustomWorkingDir     string
	Volumes              []*api.Volume
}

type Params struct {
	Image                  string
	CPU                    float32
	Memory                 uint32
	AdditionalContainers   []*api.AdditionalContainer
	CommandFrom, CommandTo string
	Platform               platform.Platform
	Architecture           *api.Architecture
	AgentVolumeName        string
	WorkingVolumeName      string
	WorkingDirectory       string
	Volumes                []*api.Volume
}

func (inst *Instance) Run(ctx context.Context, config *runconfig.RunConfig) (err error) {
	logger := config.Logger()

	containerBackend, err := config.GetContainerBackend()
	if err != nil {
		return err
	}

	agentVolume, workingVolume, err := volume.CreateWorkingVolumeFromConfig(ctx, config, inst.Platform,
		inst.Architecture)
	if err != nil {
		return err
	}
	defer func() {
		if config.ContainerOptions.NoCleanup {
			logger.Infof("not cleaning up agent volume %s, don't forget to remove it with \"docker volume rm %s\"",
				agentVolume.Name(), agentVolume.Name())
			logger.Infof("not cleaning up working volume %s, don't forget to remove it with \"docker volume rm %s\"",
				workingVolume.Name(), workingVolume.Name())

			return
		}

		cleanupErr := agentVolume.Close(containerBackend)
		if err == nil {
			err = cleanupErr
		}

		cleanupErr = workingVolume.Close(containerBackend)
		if err == nil {
			err = cleanupErr
		}
	}()

	params := &Params{
		Image:                inst.Image,
		CPU:                  inst.CPU,
		Memory:               inst.Memory,
		AdditionalContainers: inst.AdditionalContainers,
		Platform:             inst.Platform,
		Architecture:         inst.Architecture,
		AgentVolumeName:      agentVolume.Name(),
		WorkingVolumeName:    workingVolume.Name(),
		WorkingDirectory:     inst.WorkingDirectory(config.ProjectDir, config.DirtyMode),
		Volumes:              inst.Volumes,
	}

	return RunContainerizedAgent(ctx, config, params)
}

func (inst *Instance) WorkingDirectory(projectDir string, dirtyMode bool) string {
	if inst.CustomWorkingDir != "" {
		return inst.CustomWorkingDir
	}

	return inst.Platform.GenericWorkingDir()
}

func (inst *Instance) Close(context.Context) error {
	return nil
}
