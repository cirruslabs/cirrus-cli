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
			if strings.EqualFold(desiredTaskName, task.Name) {
				filteredTasks = append(filteredTasks, task)
			}
		}

		return filteredTasks
	}
}
