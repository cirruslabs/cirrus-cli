package parser

import (
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/task"
	"strings"
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
				parsererror.ErrBasic, task.Name, seenInstructionName, commandInstructionName(command))
		}

		alreadySeenNames[command.Name] = commandInstructionName(command)
	}

	return nil
}

func validateDependenciesDeep(tasks []task.ParseableTaskLike) error {
	satisfiedIDs := make(map[int64]struct{})

	for {
		// Collect tasks that still have some dependencies unsatisfied
		var unsatisfiedTasks []task.ParseableTaskLike
		for _, task := range tasks {
			if _, ok := satisfiedIDs[task.ID()]; !ok {
				unsatisfiedTasks = append(unsatisfiedTasks, task)
			}
		}

		// Try to resolve these dependencies
		var newlySatisfiedTasks []task.ParseableTaskLike
		for _, unsatisfiedTask := range unsatisfiedTasks {
			satisfied := true

			for _, dependencyID := range unsatisfiedTask.DependsOnIDs() {
				if _, ok := satisfiedIDs[dependencyID]; !ok {
					satisfied = false
					break
				}
			}

			if satisfied {
				newlySatisfiedTasks = append(newlySatisfiedTasks, unsatisfiedTask)
			}
		}

		if len(newlySatisfiedTasks) == 0 {
			// We're probably done or there's a missing/circular dependency exist
			break
		} else {
			// Remember tasks that are now resolved
			for _, newlySatisfiedTask := range newlySatisfiedTasks {
				satisfiedIDs[newlySatisfiedTask.ID()] = struct{}{}
			}
		}
	}

	// All tasks satisfied?
	if len(satisfiedIDs) != len(tasks) {
		var unsatisfiedNames []string
		for _, task := range tasks {
			if _, ok := satisfiedIDs[task.ID()]; !ok {
				unsatisfiedNames = append(unsatisfiedNames, task.Name())
			}
		}

		return fmt.Errorf("%w: error in dependencies between tasks: %v",
			parsererror.ErrBasic, strings.Join(unsatisfiedNames, ", "))
	}

	return nil
}
