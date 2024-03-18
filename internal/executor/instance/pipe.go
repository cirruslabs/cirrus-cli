package instance

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/container"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/volume"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
)

var ErrPipeCreationFailed = errors.New("failed to create pipe instance")

type PipeStage struct {
	Image                  string
	CommandFrom, CommandTo string
}

type PipeInstance struct {
	Stages           []PipeStage
	CPU              float32
	Memory           uint32
	CustomWorkingDir string
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

func (pi *PipeInstance) Run(ctx context.Context, config *runconfig.RunConfig) (err error) {
	platform := platform.NewUnix()
	logger := config.Logger()

	containerBackend, err := config.GetContainerBackend()
	if err != nil {
		return err
	}

	agentVolume, workingVolume, err := volume.CreateWorkingVolumeFromConfig(ctx, config, platform, nil)
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

	for _, stage := range pi.Stages {
		params := &container.Params{
			Image:             stage.Image,
			CPU:               pi.CPU,
			Memory:            pi.Memory,
			CommandFrom:       stage.CommandFrom,
			CommandTo:         stage.CommandTo,
			Platform:          platform,
			AgentVolumeName:   agentVolume.Name(),
			WorkingVolumeName: workingVolume.Name(),
			WorkingDirectory:  pi.WorkingDirectory(config.ProjectDir, config.DirtyMode),
		}

		if err := container.RunContainerizedAgent(ctx, config, params); err != nil {
			return err
		}
	}

	return nil
}

func (pi *PipeInstance) WorkingDirectory(projectDir string, dirtyMode bool) string {
	if pi.CustomWorkingDir != "" {
		return pi.CustomWorkingDir
	}

	return platform.NewUnix().GenericWorkingDir()
}

func (pi *PipeInstance) Close(context.Context) error {
	return nil
}
