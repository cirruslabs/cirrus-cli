package parser

import (
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
)

func commandInstructionName(command *api.Command) string {
	switch command.Instruction.(type) {
	case *api.Command_ExitInstruction:
		return "exit"
	case *api.Command_ScriptInstruction:
		return "script"
	case *api.Command_BackgroundScriptInstruction:
		return "background script"
	case *api.Command_CacheInstruction:
		return "cache"
	case *api.Command_UploadCacheInstruction:
		return "upload cache"
	case *api.Command_CloneInstruction:
		return "clone"
	case *api.Command_FileInstruction:
		return "file"
	case *api.Command_ArtifactsInstruction:
		return "artifacts"
	}

	return "unknown"
}

func validateTask(task *api.Task) error {
	alreadySeenNames := make(map[string]string)

	for _, command := range task.Commands {
		if seenInstructionName, seen := alreadySeenNames[command.Name]; seen {
			return fmt.Errorf("%w: task '%s' %s and %s instructions have identical name",
				parsererror.ErrParsing, task.Name, seenInstructionName, commandInstructionName(command))
		}

		alreadySeenNames[command.Name] = commandInstructionName(command)
	}

	return nil
}

