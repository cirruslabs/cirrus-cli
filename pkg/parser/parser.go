package parser

import (
	"context"
	"crypto/md5" // nolint:gosec // backwards compatibility
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/dummy"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/modifier/matrix"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/task"
	"github.com/golang/protobuf/ptypes"
	"github.com/lestrrat-go/jsschema"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	ErrInternal      = errors.New("internal error")
	ErrFilesContents = errors.New("failed to retrieve files contents")
)

type Parser struct {
	// Environment to take into account when expanding variables.
	environment map[string]string

	// Filesystem to reference when calculating file hashes.
	//
	// For example, Dockerfile contents are hashed to avoid duplicate builds.
	fs fs.FileSystem

	// A list of changed files useful when evaluating changesInclude() boolevator's function.
	affectedFiles []string

	boolevator *boolevator.Boolevator

	parsers             map[nameable.Nameable]parseable.Parseable
	numbering           int64
	additionalInstances map[string]protoreflect.MessageDescriptor
}

type Result struct {
	Errors []string
	Tasks  []*api.Task
}

func New(opts ...Option) *Parser {
	parser := &Parser{
		environment: make(map[string]string),
		fs:          dummy.New(),
	}

	// Apply options
	for _, opt := range opts {
		opt(parser)
	}

	// Initialize boolevator
	parser.boolevator = boolevator.New(boolevator.WithFunctions(map[string]boolevator.Function{
		"changesInclude": parser.bfuncChangesInclude(),
	}))

	// Register parsers
	parser.parsers = map[nameable.Nameable]parseable.Parseable{
		nameable.NewRegexNameable("^(.*)task$"): &task.Task{},
		nameable.NewRegexNameable("^(.*)pipe$"): &task.DockerPipe{},
	}

	return parser
}

func (p *Parser) parseTasks(tree *node.Node) ([]task.ParseableTaskLike, error) {
	var tasks []task.ParseableTaskLike

	for _, treeItem := range tree.Children {
		for key, value := range p.parsers {
			var taskLike task.ParseableTaskLike
			switch value.(type) {
			case *task.Task:
				taskLike = task.NewTask(environment.Copy(p.environment), p.boolevator, p.additionalInstances)
			case *task.DockerPipe:
				taskLike = task.NewDockerPipe(environment.Copy(p.environment), p.boolevator)
			default:
				panic("unknown task-like object")
			}

			if !key.Matches(treeItem.Name) {
				continue
			}

			err := taskLike.Parse(treeItem)
			if err != nil {
				return nil, err
			}

			// Set task's name if not set in the definition
			if taskLike.Name() == "" {
				if rn, ok := key.(*nameable.RegexNameable); ok {
					taskLike.SetName(rn.FirstGroupOrDefault(treeItem.Name, "main"))
				}
			}

			// Filtering based on "only_if" expression evaluation
			taskSpecificEnv := map[string]string{
				"CIRRUS_TASK_NAME": taskLike.Name(),
			}

			enabled, err := taskLike.Enabled(environment.Merge(taskSpecificEnv, p.environment), p.boolevator)
			if err != nil {
				return nil, err
			}

			if !enabled {
				continue
			}

			tasks = append(tasks, taskLike)
		}
	}

	return tasks, nil
}

func (p *Parser) Parse(ctx context.Context, config string) (*Result, error) {
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
	tasks, err := p.parseTasks(tree)
	if err != nil {
		return &Result{
			Errors: []string{err.Error()},
		}, nil
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
	serviceTasks, err := p.createServiceTasks(ctx, protoTasks)
	if err != nil {
		return &Result{
			Errors: []string{err.Error()},
		}, nil
	}
	protoTasks = append(protoTasks, serviceTasks...)

	// Postprocess individual tasks
	for _, protoTask := range protoTasks {
		// Insert empty clone instruction if custom clone script wasn't provided by the user
		ensureCloneInstruction(protoTask)

		// Provide unique labels for identically named tasks
		uniqueLabelsForTask, err := uniqueLabels(protoTask, protoTasks)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInternal, err)
		}
		protoTask.Metadata.UniqueLabels = uniqueLabelsForTask
	}

	return &Result{
		Tasks: protoTasks,
	}, nil
}

func (p *Parser) ParseFromFile(ctx context.Context, path string) (*Result, error) {
	config, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return p.Parse(ctx, string(config))
}

func (p *Parser) fileHash(ctx context.Context, path string, additionalBytes []byte) (string, error) {
	// Note that this will be empty if we don't know anything about the file,
	// so we'll return MD5(""), but that's OK since the purpose is caching
	fileBytes, err := p.fs.Get(ctx, path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	// nolint:gosec // backwards compatibility
	digest := md5.Sum(append(fileBytes, additionalBytes...))

	return fmt.Sprintf("%x", digest), nil
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

func (p *Parser) createServiceTask(
	dockerfileHash string,
	protoTask *api.Task,
	taskContainer *api.ContainerInstance,
) (*api.Task, error) {
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

	// Craft Docker build arguments: task name
	var buildArgsSlice []string
	for key, value := range taskContainer.DockerArguments {
		buildArgsSlice = append(buildArgsSlice, fmt.Sprintf("%s=%s", key, value))
	}
	sort.Strings(buildArgsSlice)
	var buildArgs string
	for _, buildArg := range buildArgsSlice {
		buildArgs += fmt.Sprintf(" %s", buildArg)
	}

	// Craft Docker build arguments: docker build command
	var dockerBuildArgsSlice []string
	for key, value := range taskContainer.DockerArguments {
		dockerBuildArgsSlice = append(dockerBuildArgsSlice, fmt.Sprintf("%s=\"%s\"", key, value))
	}
	sort.Strings(dockerBuildArgsSlice)
	var dockerBuildArgs string
	for _, dockerBuildArg := range dockerBuildArgsSlice {
		dockerBuildArgs += fmt.Sprintf(" --build-arg %s", dockerBuildArg)
	}

	serviceTask := &api.Task{
		Name:         fmt.Sprintf("Prebuild %s%s", taskContainer.DockerfilePath, buildArgs),
		LocalGroupId: p.NextTaskID(),
		Instance:     anyInstance,
		Commands: []*api.Command{
			{
				Name: "build",
				Instruction: &api.Command_ScriptInstruction{
					ScriptInstruction: &api.ScriptInstruction{
						Scripts: []string{fmt.Sprintf("docker build "+
							"--tag gcr.io/%s:%s "+
							"--file %s%s "+
							"${CIRRUS_DOCKER_CONTEXT:-$CIRRUS_WORKING_DIR}",
							prebuiltInstance.Repository, prebuiltInstance.Reference,
							taskContainer.DockerfilePath, dockerBuildArgs)},
					},
				},
			},
			{
				Name: "push",
				Instruction: &api.Command_ScriptInstruction{
					ScriptInstruction: &api.ScriptInstruction{
						Scripts: []string{fmt.Sprintf("gcloud docker -- push gcr.io/cirrus-ci-community/%s:latest",
							dockerfileHash)},
					},
				},
			},
		},
		Environment: protoTask.Environment,
		Metadata: &api.Task_Metadata{
			Properties: task.DefaultTaskProperties(),
		},
	}

	// Some metadata property fields duplicate other fields
	serviceTask.Metadata.Properties["indexWithinBuild"] = strconv.FormatInt(serviceTask.LocalGroupId, 10)

	// Some metadata property fields are preserved from the original task
	serviceTask.Metadata.Properties["timeoutInSeconds"] = protoTask.Metadata.Properties["timeoutInSeconds"]

	return serviceTask, nil
}

func (p *Parser) createServiceTasks(ctx context.Context, protoTasks []*api.Task) ([]*api.Task, error) {
	serviceTasks := make(map[string]*api.Task)

	for _, protoTask := range protoTasks {
		var dynamicInstance ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(protoTask.Instance, &dynamicInstance); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInternal, err)
		}

		taskContainer, ok := dynamicInstance.Message.(*api.ContainerInstance)
		if !ok {
			continue
		}

		if taskContainer.Platform != api.Platform_LINUX {
			return nil, fmt.Errorf("%w: unsupported platform for building Dockerfile: %s",
				parsererror.ErrParsing, taskContainer.Platform.String())
		}

		if taskContainer.DockerfilePath == "" {
			continue
		}

		// Craft Docker build arguments: arguments used in content hash calculation
		var hashableArgsSlice []string
		for key, value := range taskContainer.DockerArguments {
			hashableArgsSlice = append(hashableArgsSlice, key+value)
		}
		sort.Strings(hashableArgsSlice)
		hashableArgs := strings.Join(hashableArgsSlice, ", ")

		dockerfileHash, err := p.fileHash(ctx, taskContainer.DockerfilePath, []byte(hashableArgs))
		if err != nil {
			return nil, err
		}

		// Find or create service task
		serviceTaskKey := taskContainer.DockerfilePath + hashableArgs

		serviceTask, ok := serviceTasks[serviceTaskKey]
		if !ok {
			serviceTask, err = p.createServiceTask(dockerfileHash, protoTask, taskContainer)
			if err != nil {
				return nil, err
			}

			serviceTasks[serviceTaskKey] = serviceTask
		}

		// Set dependency to the found or created service task
		protoTask.RequiredGroups = append(protoTask.RequiredGroups, serviceTask.LocalGroupId)

		// Ensure that the task will use our to-be-created image
		taskContainer.Image = fmt.Sprintf("gcr.io/cirrus-ci-community/%s:latest", dockerfileHash)
		updatedInstance, err := ptypes.MarshalAny(taskContainer)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInternal, err)
		}
		protoTask.Instance = updatedInstance
	}

	// Extract map values to a slice
	var result []*api.Task
	for _, serviceTask := range serviceTasks {
		result = append(result, serviceTask)
	}

	return result, nil
}

func ensureCloneInstruction(task *api.Task) {
	for _, command := range task.Commands {
		if command.Name == "clone" {
			return
		}
	}

	// Inherit "image" property from the first task (if any),
	// or otherwise we might break Docker Pipe
	var properties map[string]string

	if len(task.Commands) != 0 {
		image, ok := task.Commands[0].Properties["image"]
		if ok {
			properties = map[string]string{
				"image": image,
			}
			delete(task.Commands[0].Properties, "image")
		}
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
