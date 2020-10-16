package parser

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"google.golang.org/protobuf/reflect/protoregistry"
	"sort"
)

func labels(task *api.Task) ([]string, error) {
	labels, err := instanceLabels(task.Instance)
	if err != nil {
		return labels, err
	}

	// Environment-specific labels
	for key, value := range task.Environment {
		labels = append(labels, fmt.Sprintf("%s:%s", key, value))
	}

	return labels, nil
}

func instanceLabels(taskInstance *any.Any) ([]string, error) {
	// Instance-specific labels
	var labels []string

	// Unmarshal instance
	var dynamicAny ptypes.DynamicAny

	err := ptypes.UnmarshalAny(taskInstance, &dynamicAny)

	if errors.Is(err, protoregistry.NotFound) {
		return labels, nil
	}

	if err != nil {
		return nil, err
	}

	switch instance := dynamicAny.Message.(type) {
	case *api.ContainerInstance:
		if instance.DockerfilePath == "" {
			labels = append(labels, fmt.Sprintf("container:%s", instance.Image))

			if instance.OperationSystemVersion != "" {
				labels = append(labels, fmt.Sprintf("os:%s", instance.OperationSystemVersion))
			}
		} else {
			labels = append(labels, fmt.Sprintf("Dockerfile:%s", instance.DockerfilePath))

			for key, value := range instance.DockerArguments {
				labels = append(labels, fmt.Sprintf("%s:%s", key, value))
			}
		}

		for _, additionalContainer := range instance.AdditionalContainers {
			labels = append(labels, fmt.Sprintf("%s:%s", additionalContainer.Name, additionalContainer.Image))
		}
	case *api.PipeInstance:
		labels = append(labels, "pipe")
	}
	return labels, nil
}

func uniqueLabels(task *api.Task, tasks []*api.Task) ([]string, error) {
	// Collect similarly named tasks, including the task itself
	var similarlyNamedTasks []*api.Task

	for _, protoTask := range tasks {
		if task.Name == protoTask.Name {
			similarlyNamedTasks = append(similarlyNamedTasks, protoTask)
		}
	}

	// No need to set any labels in case there are no similarly named tasks (except the task itself)
	if len(similarlyNamedTasks) == 1 {
		return []string{}, nil
	}

	// Collect labels that are common to all similarly named tasks
	commonLabels := make(map[string]int)

	for _, similarlyNamedTask := range similarlyNamedTasks {
		labels, err := labels(similarlyNamedTask)
		if err != nil {
			return nil, err
		}

		for _, label := range labels {
			commonLabels[label]++
		}
	}

	for key, value := range commonLabels {
		if value != len(similarlyNamedTasks) {
			delete(commonLabels, key)
		}
	}

	// Get labels specific for this task
	labels, err := labels(task)
	if err != nil {
		return nil, err
	}

	// Remove common labels
	var keptLabels []string

	for _, label := range labels {
		if _, ok := commonLabels[label]; !ok {
			keptLabels = append(keptLabels, label)
		}
	}

	labels = keptLabels

	// Sort labels to make them comparable
	sort.Strings(labels)

	return labels, nil
}
