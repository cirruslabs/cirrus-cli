package taskfilter

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"strings"
)

type TaskFilter func([]*api.Task) []*api.Task

func MatchAnyTask() TaskFilter {
	return func(tasks []*api.Task) []*api.Task {
		return tasks
	}
}

func MatchExactTask(desiredTaskName string) TaskFilter {
	return func(tasks []*api.Task) []*api.Task {
		var filteredTasks []*api.Task

		for _, task := range tasks {
			// Ensure that this task's name matches with the name we're looking for
			desiredTaskNameLower := strings.ToLower(desiredTaskName)
			taskNameLower := strings.ToLower(task.Name)

			if !strings.HasPrefix(desiredTaskNameLower, taskNameLower) {
				continue
			}

			// In case we're looking for a task with specific labels — extract them and ensure they all match
			desiredLabels := extractLabels(strings.TrimPrefix(desiredTaskNameLower, taskNameLower))

			var actualLabels []string
			if task.Metadata != nil {
				actualLabels = task.Metadata.UniqueLabels
			}

			if !containsAll(desiredLabels, actualLabels) {
				continue
			}

			// Clear task's dependencies
			task.RequiredGroups = task.RequiredGroups[:0]

			filteredTasks = append(filteredTasks, task)
		}

		return filteredTasks
	}
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

func containsAll(desiredLabels, actualLabels []string) bool {
	var numMatchedLabels int

	for _, soughtLabel := range desiredLabels {
		for _, actualLabel := range actualLabels {
			if strings.EqualFold(soughtLabel, actualLabel) {
				numMatchedLabels++
				break
			}
		}
	}

	return numMatchedLabels == len(desiredLabels)
}
