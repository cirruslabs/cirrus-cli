package parser

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
	"sort"
	"strings"
)

func labels(
	task *api.Task,
	additionalInstances map[string]protoreflect.MessageDescriptor,
) ([]string, error) {
	labels, err := instanceLabels(task.Instance, additionalInstances)
	if err != nil {
		return labels, err
	}

	// Environment-specific labels
	for key, value := range task.Environment {
		labels = append(labels, fmt.Sprintf("%s:%s", key, value))
	}

	return labels, nil
}

func instanceLabels(
	taskInstance *any.Any,
	additionalInstances map[string]protoreflect.MessageDescriptor,
) ([]string, error) {
	for instanceName, descriptor := range additionalInstances {
		if strings.HasSuffix(taskInstance.GetTypeUrl(), string(descriptor.FullName())) {
			return extractProtoInstanceLabels(taskInstance, instanceName, descriptor)
		}
	}
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

func extractProtoInstanceLabels(anyInstance *any.Any, instanceName string, descriptor protoreflect.MessageDescriptor) ([]string, error) {
	dynamicInstance := dynamicpb.NewMessage(descriptor)

	var instanceLabels []string

	err := proto.Unmarshal(anyInstance.Value, dynamicInstance)

	if err != nil {
		return nil, err
	}

	if fieldDescriptor := descriptor.Fields().ByName("image"); fieldDescriptor != nil {
		instanceLabels = append(instanceLabels, fmt.Sprintf("%s:%s", instanceName, dynamicInstance.Get(fieldDescriptor)))
	} else if fieldDescriptor := descriptor.Fields().ByName("image_name"); fieldDescriptor != nil {
		instanceLabels = append(instanceLabels, fmt.Sprintf("%s:%s", instanceName, dynamicInstance.Get(fieldDescriptor)))
	} else if fieldDescriptor := descriptor.Fields().ByName("image_id"); fieldDescriptor != nil {
		instanceLabels = append(instanceLabels, fmt.Sprintf("%s:%s", instanceName, dynamicInstance.Get(fieldDescriptor)))
	} else if fieldDescriptor := descriptor.Fields().ByName("template"); fieldDescriptor != nil {
		instanceLabels = append(instanceLabels, fmt.Sprintf("%s:%s", instanceName, dynamicInstance.Get(fieldDescriptor)))
	}

	if fieldDescriptor := descriptor.Fields().ByName("dockerfilePath"); fieldDescriptor != nil {
		path := dynamicInstance.Get(fieldDescriptor).String()
		if path != "" {
			instanceLabels = append(instanceLabels, fmt.Sprintf("Dockerfile:%s", path))
		}
	}

	if fieldDescriptor := descriptor.Fields().ByName("dockerArguments"); fieldDescriptor != nil {
		dynamicInstance.Get(fieldDescriptor).Map().Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
			instanceLabels = append(instanceLabels, fmt.Sprintf("%s:%s", key, value))
			return true
		})
	}

	if fieldDescriptor := descriptor.Fields().ByName("zone"); fieldDescriptor != nil {
		zone := dynamicInstance.Get(fieldDescriptor).String()
		if zone != "" {
			instanceLabels = append(instanceLabels, fmt.Sprintf("zone:%s", zone))
		}
	}
	return instanceLabels, nil
}

func uniqueLabels(task *api.Task, tasks []*api.Task, additionalInstances map[string]protoreflect.MessageDescriptor) ([]string, error) {
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
		labels, err := labels(similarlyNamedTask, additionalInstances)
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
	labels, err := labels(task, additionalInstances)
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
