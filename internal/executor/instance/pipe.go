package instance

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"path"
)

var ErrPipeCreationFailed = errors.New("failed to create pipe instance")

type PipeStage struct {
	Image                  string
	CommandFrom, CommandTo string
}

type PipeInstance struct {
	Stages []PipeStage
	CPU    float32
	Memory uint32
}

// PipeStagesFromCommands uses image hints in commands to build the stages.
func PipeStagesFromCommands(commands []*api.Command) ([]PipeStage, error) {
	var stages []PipeStage

	for i, command := range commands {
		image, found := command.Properties["image"]
		if !found {
			if i == 0 {
				return nil, fmt.Errorf("%w: first command does not have an image property", ErrPipeCreationFailed)
			}

			continue
		}

		// Close old stage
		if len(stages) != 0 {
			stages[len(stages)-1].CommandTo = command.Name
		}

		// Open new stage
		stages = append(stages, PipeStage{
			Image:       image,
			CommandFrom: command.Name,
		})
	}

	return stages, nil
}

func (pi *PipeInstance) Run(ctx context.Context, config *RunConfig) (err error) {
	platform := platform.NewUnix()

	workingVolume, err := CreateWorkingVolumeFromConfig(ctx, config, platform)
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

	for _, stage := range pi.Stages {
		params := &Params{
			Image:             stage.Image,
			CPU:               pi.CPU,
			Memory:            pi.Memory,
			CommandFrom:       stage.CommandFrom,
			CommandTo:         stage.CommandTo,
			Platform:          platform,
			WorkingVolumeName: workingVolume.Name(),
		}

		if err := RunContainerizedAgent(ctx, config, params); err != nil {
			return err
		}
	}

	return nil
}

func (pi *PipeInstance) WorkingDirectory(projectDir string, dirtyMode bool) string {
	return path.Join(platform.NewUnix().WorkingVolumeMountpoint(), platform.WorkingVolumeWorkingDir)
}
