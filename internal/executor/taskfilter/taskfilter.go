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

func MatchExactTask(desiredTaskName string) TaskFilter {
	return func(tasks []*api.Task) ([]*api.Task, error) {
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

			if !containsAll(actualLabels, desiredLabels) {
				continue
			}

			// Clear task's dependencies
			task.RequiredGroups = task.RequiredGroups[:0]

			filteredTasks = append(filteredTasks, task)
		}

		if len(filteredTasks) == 0 {
			return nil, fmt.Errorf("%w: none of the %d task(s) were matched using a %q filter",
				ErrNoMatch, len(tasks), desiredTaskName)
		}

		return filteredTasks, nil
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
