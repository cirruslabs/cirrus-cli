package parser

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/modifier/matrix"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/task"
	"github.com/golang/protobuf/ptypes"
	"github.com/lestrrat-go/jsschema"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
	"regexp"
)

var (
	ErrInternal      = errors.New("internal error")
	ErrFilesContents = errors.New("failed to retrieve files contents")
)

type Parser struct {
	// Environment to take into account when expanding variables.
	environment map[string]string

	// Paths and contents of the files that might influence the parser.
	filesContents map[string]string

	parsers   map[nameable.Nameable]parseable.Parseable
	numbering int64
}

type Result struct {
	Errors []string
	Tasks  []*api.Task
}

func New(opts ...Option) *Parser {
	parser := &Parser{
		environment:   make(map[string]string),
		filesContents: make(map[string]string),
	}

	// Apply options
	for _, opt := range opts {
		opt(parser)
	}

	// Register parsers
	parser.parsers = map[nameable.Nameable]parseable.Parseable{
		nameable.NewRegexNameable("(.*)task"): &task.Task{},
		nameable.NewRegexNameable("(.*)pipe"): &task.DockerPipe{},
	}

	return parser
}

func (p *Parser) Parse(config string) (*Result, error) {
	var parsed yaml.MapSlice

	// Unmarshal YAML
	if err := yaml.Unmarshal([]byte(config), &parsed); err != nil {
		return nil, err
	}

	// Run modifiers on it
	modified, err := matrix.ExpandMatrices(parsed)
	if err != nil {
		return nil, err
	}

	// Convert the parsed and nested YAML structure into a tree
	// to get the ability to walk parents
	tree, err := node.NewFromSlice(modified)
	if err != nil {
		return nil, err
	}

	// Run parsers on the top-level nodes
	var tasks []task.ParseableTaskLike

	for _, treeItem := range tree.Children {
		for key, value := range p.parsers {
			var taskLike task.ParseableTaskLike
			switch value.(type) {
			case *task.Task:
				taskLike = task.NewTask(environment.Copy(p.environment))
			case *task.DockerPipe:
				taskLike = task.NewDockerPipe(environment.Copy(p.environment))
			default:
				panic("unknown task-like object")
			}

			if !key.Matches(treeItem.Name) {
				continue
			}

			err := taskLike.Parse(treeItem)
			if err != nil {
				return &Result{
					Errors: []string{err.Error()},
				}, nil
			}

			// Set task's name if not set in the definition
			if taskLike.Name() == "" {
				if rn, ok := key.(*nameable.RegexNameable); ok {
					taskLike.SetName(rn.FirstGroupOrDefault(treeItem.Name, "main"))
				}
			}

			// Filtering based on "only_if" expression evaluation
			if !taskLike.Enabled() {
				continue
			}

			tasks = append(tasks, taskLike)
		}
	}

	// Assign group IDs to tasks
	for _, task := range tasks {
		task.SetID(p.NextTaskID())
	}

	// Resolve dependencies
	if err := resolveDependencies(tasks); err != nil {
		return &Result{
			Errors: []string{err.Error()},
		}, nil
	}

	if len(tasks) == 0 {
		return &Result{
			Errors: []string{"configuration was parsed without errors, but no tasks were found"},
		}, nil
	}

	var protoTasks []*api.Task
	for _, task := range tasks {
		protoTasks = append(protoTasks, task.Proto().(*api.Task))
	}

	// Create service tasks
	serviceTasks, err := p.createServiceTasks(protoTasks)
	if err != nil {
		return &Result{
			Errors: []string{err.Error()},
		}, nil
	}
	protoTasks = append(protoTasks, serviceTasks...)

	// Final pass over resulting tasks in Protocol Buffers format
	for _, protoTask := range protoTasks {
		// Insert empty clone instruction if custom clone script wasn't provided by the user
		ensureCloneInstruction(protoTask)

		// Provide unique labels for identically named tasks
		if countTasksWithName(protoTasks, protoTask.Name) > 1 {
			if err := populateUniqueLabels(protoTask); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrInternal, err)
			}
		}
	}

	return &Result{
		Tasks: protoTasks,
	}, nil
}

func (p *Parser) ParseFromFile(path string) (*Result, error) {
	config, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	result, err := p.Parse(string(config))
	if err != nil || len(result.Errors) != 0 {
		return result, err
	}

	// Get the contents of files that might influence the parser results
	//
	// For example, when using Dockerfile as CI environment feature[1], the unique hash of the container
	// image is calculated from the file specified in the "dockerfile" field.
	//
	// [1]: https://cirrus-ci.org/guide/docker-builder-vm/#dockerfile-as-a-ci-environment
	filesContents := make(map[string]string)
	for _, task := range result.Tasks {
		inst, err := instance.NewFromProto(task.Instance, []*api.Command{})
		if err != nil {
			continue
		}
		prebuilt, ok := inst.(*instance.PrebuiltInstance)
		if !ok {
			continue
		}
		contents, err := ioutil.ReadFile(filepath.Join(filepath.Dir(path), prebuilt.Dockerfile))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrFilesContents, err)
		}
		filesContents[prebuilt.Dockerfile] = string(contents)
	}

	// Short-circuit if we've found no special files
	if len(filesContents) == 0 {
		return result, nil
	}

	// Parse again with the file contents supplied
	p.filesContents = filesContents
	return p.Parse(string(config))
}

func (p *Parser) ContentHash(filePath string) string {
	// Note that this will be empty if we don't know anything about the file,
	// so we'll return SHA256(""), but that's OK since the purpose is caching
	fileContents := p.filesContents[filePath]

	return fmt.Sprintf("%x", sha256.Sum256([]byte(fileContents)))
}

func (p *Parser) NextTaskID() int64 {
	defer func() {
		p.numbering++
	}()
	return p.numbering
}

func (p *Parser) Schema() *schema.Schema {
	schema := &schema.Schema{
		Properties:        make(map[string]*schema.Schema),
		PatternProperties: make(map[*regexp.Regexp]*schema.Schema),
	}

	schema.ID = "https://cirrus-ci.org/"
	schema.Title = "JSON schema for Cirrus CI configuration files"

	// Apply parser schemas
	for key, value := range p.parsers {
		switch keyNameable := key.(type) {
		case *nameable.SimpleNameable:
			schema.Properties[keyNameable.Name()] = value.Schema()
		case *nameable.RegexNameable:
			schema.PatternProperties[keyNameable.Regex()] = value.Schema()
		}
	}

	// Apply field schemas

	return schema
}

func resolveDependencies(tasks []task.ParseableTaskLike) error {
	for _, task := range tasks {
		var dependsOnIDs []int64
		for _, dependsOnName := range task.DependsOnNames() {
			var foundID int64 = -1
			for _, subTask := range tasks {
				if subTask.Name() == dependsOnName {
					foundID = subTask.ID()
					break
				}
			}
			if foundID == -1 {
				return fmt.Errorf("%w: dependency not found", parsererror.ErrParsing)
			}
			dependsOnIDs = append(dependsOnIDs, foundID)
		}
		task.SetDependsOnIDs(dependsOnIDs)
	}

	return nil
}

func (p *Parser) createServiceTasks(protoTasks []*api.Task) ([]*api.Task, error) {
	var serviceTasks []*api.Task

	for _, task := range protoTasks {
		var dynamicInstance ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(task.Instance, &dynamicInstance); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInternal, err)
		}

		taskContainer, ok := dynamicInstance.Message.(*api.ContainerInstance)
		if !ok {
			continue
		}

		if taskContainer.DockerfilePath == "" {
			continue
		}

		dockerfileHash := p.ContentHash(taskContainer.DockerfilePath)

		prebuiltInstance := &api.PrebuiltImageInstance{
			Repository:     fmt.Sprintf("cirrus-ci-community/%s", dockerfileHash),
			Reference:      "latest",
			Platform:       taskContainer.Platform,
			DockerfilePath: taskContainer.DockerfilePath,
			Arguments:      taskContainer.DockerArguments,
		}

		anyInstance, err := ptypes.MarshalAny(prebuiltInstance)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInternal, err)
		}

		newTask := &api.Task{
			Name:         fmt.Sprintf("Prebuild %s", taskContainer.DockerfilePath),
			LocalGroupId: p.NextTaskID(),
			Instance:     anyInstance,
			Commands: []*api.Command{
				{
					Name: "dummy",
					Instruction: &api.Command_ScriptInstruction{
						ScriptInstruction: &api.ScriptInstruction{
							Scripts: []string{"true"},
						},
					},
				},
			},
		}

		serviceTasks = append(serviceTasks, newTask)

		task.RequiredGroups = append(task.RequiredGroups, newTask.LocalGroupId)

		// Ensure that the task will use our to-be-created image
		taskContainer.Image = fmt.Sprintf("gcr.io/cirrus-ci-community/%s:latest", dockerfileHash)
		updatedInstance, err := ptypes.MarshalAny(taskContainer)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInternal, err)
		}
		task.Instance = updatedInstance
	}

	return serviceTasks, nil
}

func ensureCloneInstruction(task *api.Task) {
	if len(task.Commands) == 0 {
		return
	}

	for _, command := range task.Commands {
		if command.Name == "clone" {
			return
		}
	}

	// Inherit "image" property from the first task (if any),
	// or otherwise we might break Docker Pipe
	var properties map[string]string
	image, ok := task.Commands[0].Properties["image"]
	if ok {
		properties = map[string]string{
			"image": image,
		}
		delete(task.Commands[0].Properties, "image")
	}

	cloneCommand := &api.Command{
		Name: "clone",
		Instruction: &api.Command_CloneInstruction{
			CloneInstruction: &api.CloneInstruction{},
		},
		Properties: properties,
	}

	task.Commands = append([]*api.Command{cloneCommand}, task.Commands...)
}

func countTasksWithName(protoTasks []*api.Task, name string) (result int) {
	for _, protoTask := range protoTasks {
		if protoTask.Name == name {
			result++
		}
	}

	return
}

func populateUniqueLabels(task *api.Task) error {
	// Unmarshal instance
	var dynamicAny ptypes.DynamicAny

	if err := ptypes.UnmarshalAny(task.Instance, &dynamicAny); err != nil {
		return err
	}

	// Populate labels
	var labels []string

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

	// Update task
	task.Metadata.UniqueLabels = labels

	return nil
}
