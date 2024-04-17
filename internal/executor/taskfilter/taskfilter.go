package taskfilter

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"strings"
)

var ErrNoMatch = errors.New("task filter yielded no matches")

type TaskFilter func([]*api.Task) ([]*api.Task, error)

func MatchAnyTask() TaskFilter {
	return func(tasks []*api.Task) ([]*api.Task, error) {
		return tasks, nil
	}
}

func MatchExactTask(desiredTaskNameOrAlias string) TaskFilter {
	return func(tasks []*api.Task) ([]*api.Task, error) {
		var matchedTasks []*api.Task

		for _, task := range tasks {
			// Ensure that this task's name (or an alias) matches
			// with the name (or an alias) that we're looking for
			if !matchTask(desiredTaskNameOrAlias, task.Name, task.Metadata) &&
				!matchTask(desiredTaskNameOrAlias, taskAlias(task), task.Metadata) {
				continue
			}

			// Clear the task's dependencies
			task.RequiredGroups = task.RequiredGroups[:0]

			matchedTasks = append(matchedTasks, task)
		}

		if len(matchedTasks) == 0 {
			return nil, fmt.Errorf("%w: none of the %d task(s) were matched using a %q filter",
				ErrNoMatch, len(tasks), desiredTaskNameOrAlias)
		}

		return matchedTasks, nil
	}
}

func taskAlias(task *api.Task) string {
	if task.Metadata == nil {
		return ""
	}

	if task.Metadata.Properties == nil {
		return ""
	}

	return task.Metadata.Properties["alias"]
}

func matchTask(desiredTaskName string, nameOrAlias string, metadata *api.Task_Metadata) bool {
	if nameOrAlias == "" {
		return false
	}

	desiredTaskNameLower := strings.ToLower(desiredTaskName)
	nameOrAliasLower := strings.ToLower(nameOrAlias)

	if !strings.HasPrefix(desiredTaskNameLower, nameOrAliasLower) {
		return false
	}

	// In case we're looking for a task with specific labels — extract them and ensure they all match
	desiredLabels := extractLabels(strings.TrimPrefix(desiredTaskNameLower, nameOrAliasLower))

	var actualLabels []string
	if metadata != nil {
		actualLabels = metadata.UniqueLabels
	}

	return containsAll(actualLabels, desiredLabels)
}

func extractLabels(s string) (result []string) {
	labels := strings.Split(s, " ")

	// Filter out empty labels
	for _, label := range labels {
		if strings.TrimSpace(label) == "" {
			continue
		}

		result = append(result, label)
	}

	return
}

func containsAll(actualLabels []string, desiredLabels []string) bool {
	var numMatchedLabels int

	for _, desiredLabel := range desiredLabels {
		for _, actualLabel := range actualLabels {
			if strings.EqualFold(desiredLabel, actualLabel) {
				numMatchedLabels++
				break
			}
		}
	}

	return numMatchedLabels == len(desiredLabels)
}
