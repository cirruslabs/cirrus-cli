package parser

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"sort"
	"strings"
)

func (p *Parser) labels(
	task *api.Task,
	additionalInstances map[string]protoreflect.MessageDescriptor,
) ([]string, error) {
	if task.Instance == nil && p.missingInstancesAllowed {
		return []string{}, nil
	}

	labels, err := instanceLabels(task.Instance, additionalInstances)
	if err != nil {
		return labels, err
	}

	// Environment-specific labels
	for key, value := range task.Environment {
		if strings.HasPrefix(value, "ENCRYPTED[") && strings.HasSuffix(value, "]") {
			continue
		}

		labels = append(labels, fmt.Sprintf("%s:%s", key, value))
	}

	return labels, nil
}

func instanceLabels(
	taskInstance *anypb.Any,
	additionalInstances map[string]protoreflect.MessageDescriptor,
) ([]string, error) {
	// Provide stable iteration order
	var sortedInstanceKeys []string
	for key := range additionalInstances {
		sortedInstanceKeys = append(sortedInstanceKeys, key)
	}
	sort.Strings(sortedInstanceKeys)

	for _, instanceName := range sortedInstanceKeys {
		descriptor := additionalInstances[instanceName]
		if strings.HasSuffix(taskInstance.GetTypeUrl(), string(descriptor.FullName())) {
			return extractProtoInstanceLabels(taskInstance, instanceName, descriptor)
		}
	}

	// Instance-specific labels
	var labels []string

	// Unmarshal instance
	dynamicAny, err := taskInstance.UnmarshalNew()

	if errors.Is(err, protoregistry.NotFound) {
		return labels, nil
	}

	if err != nil {
		return nil, err
	}

	switch instance := dynamicAny.(type) {
	case *api.ContainerInstance:
		if instance.Dockerfile == "" {
			labels = append(labels, fmt.Sprintf("container:%s", instance.Image))

			if instance.OsVersion != "" {
				labels = append(labels, fmt.Sprintf("os:%s", instance.OsVersion))
			}
		} else {
			labels = append(labels, fmt.Sprintf("Dockerfile:%s", instance.Dockerfile))

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

	// Filter out labels with empty values
	var nonEmptyLabels []string
	for _, label := range labels {
		fmt.Println(label)

		if strings.HasSuffix(label, ":") {
			continue
		}

		nonEmptyLabels = append(nonEmptyLabels, label)
	}

	return nonEmptyLabels, nil
}

func extractProtoInstanceLabels(
	anyInstance *anypb.Any,
	instanceName string,
	descriptor protoreflect.MessageDescriptor,
) ([]string, error) {
	dynamicInstance := dynamicpb.NewMessage(descriptor)

	var instanceLabels []string

	err := proto.Unmarshal(anyInstance.Value, dynamicInstance)

	if err != nil {
		return nil, err
	}

	instanceValue := ""
	//nolint:nestif
	if fd := descriptor.Fields().ByName("container"); checkFieldIsSet(dynamicInstance, fd) {
		instanceValue = dynamicInstance.Get(fd).String()
	} else if fd := descriptor.Fields().ByName("image_family"); checkFieldIsSet(dynamicInstance, fd) {
		instanceValue = "family/" + dynamicInstance.Get(fd).String()
	} else if fd := descriptor.Fields().ByName("image"); checkFieldIsSet(dynamicInstance, fd) {
		instanceValue = dynamicInstance.Get(fd).String()
	} else if fd := descriptor.Fields().ByName("image_name"); checkFieldIsSet(dynamicInstance, fd) {
		instanceValue = dynamicInstance.Get(fd).String()
	} else if fd := descriptor.Fields().ByName("image_id"); checkFieldIsSet(dynamicInstance, fd) {
		instanceValue = dynamicInstance.Get(fd).String()
	} else if fd := descriptor.Fields().ByName("template"); checkFieldIsSet(dynamicInstance, fd) {
		instanceValue = dynamicInstance.Get(fd).String()
	}
	if instanceValue != "" {
		instanceLabels = append(instanceLabels, fmt.Sprintf("%s:%s", instanceName, instanceValue))
	}

	if fd := descriptor.Fields().ByName("dockerfile"); checkFieldIsSet(dynamicInstance, fd) {
		instanceLabels = append(instanceLabels, fmt.Sprintf("Dockerfile:%s", dynamicInstance.Get(fd).String()))
	}

	if fd := descriptor.Fields().ByName("docker_arguments"); checkFieldIsSet(dynamicInstance, fd) {
		dynamicInstance.Get(fd).Map().Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
			instanceLabels = append(instanceLabels, fmt.Sprintf("%s:%s", key, value))
			return true
		})
	}

	return instanceLabels, nil
}

func checkFieldIsSet(dynamicInstance *dynamicpb.Message, fd protoreflect.FieldDescriptor) bool {
	return fd != nil && dynamicInstance.Has(fd)
}

func (p *Parser) uniqueLabels(
	task *api.Task,
	tasks []*api.Task,
	additionalInstances map[string]protoreflect.MessageDescriptor,
) ([]string, error) {
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
		labels, err := p.labels(similarlyNamedTask, additionalInstances)
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

	// Get labels specific for this task's instance
	labels, err := p.labels(task, additionalInstances)
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
