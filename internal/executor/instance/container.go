package instance

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"path"
)

type ContainerInstance struct {
	Image                string
	CPU                  float32
	Memory               uint32
	AdditionalContainers []*api.AdditionalContainer
}

func (inst *ContainerInstance) Run(ctx context.Context, config *RunConfig) (err error) {
	workingVolume, err := CreateWorkingVolumeFromConfig(ctx, config)
	if err != nil {
		return err
	}
	defer func() {
		if config.ContainerOptions.NoCleanup {
			config.Logger.Infof("not cleaning up working volume %s, don't forget to remove it with \"docker volume rm %s\"",
				workingVolume.Name(), workingVolume.Name())

			return
		}

		cleanupErr := workingVolume.Close(config.ContainerBackend)
		if err == nil {
			err = cleanupErr
		}
	}()

	params := &Params{
		Image:                inst.Image,
		CPU:                  inst.CPU,
		Memory:               inst.Memory,
		AdditionalContainers: inst.AdditionalContainers,
		WorkingVolumeName:    workingVolume.Name(),
	}

	if err := RunContainerizedAgent(ctx, config, params); err != nil {
		return err
	}

	return nil
}

func (inst *ContainerInstance) WorkingDirectory(projectDir string, dirtyMode bool) string {
	return path.Join(WorkingVolumeMountpoint, WorkingVolumeWorkingDir)
}
