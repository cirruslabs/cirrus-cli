package instance

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
)

type ContainerInstance struct {
	Image                string
	CPU                  float32
	Memory               uint32
	AdditionalContainers []*api.AdditionalContainer
}

func (inst *ContainerInstance) Run(ctx context.Context, config *RunConfig) error {
	workingVolume, err := CreateWorkingVolumeFromConfig(ctx, config)
	if err != nil {
		return err
	}
	defer workingVolume.Close()

	params := &Params{
		Image:                inst.Image,
		CPU:                  inst.CPU,
		Memory:               inst.Memory,
		AdditionalContainers: inst.AdditionalContainers,
		WorkingVolumeName:    workingVolume.Name(),
	}

	if err := RunDockerizedAgent(ctx, config, params); err != nil {
		return err
	}

	return nil
}
