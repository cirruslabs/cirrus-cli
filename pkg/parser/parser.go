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
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/lestrrat-go/jsschema"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const pathYAML = ".cirrus.yml"

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

	parsers                  map[nameable.Nameable]parseable.Parseable
	idNumbering              int64
	indexNumbering           int64
	additionalInstances      map[string]protoreflect.MessageDescriptor
	additionalTaskProperties []*descriptor.FieldDescriptorProto
	missingInstancesAllowed  bool

	tasksCountBeforeFiltering   int64
	disabledTaskNamesAndAliases map[string]struct{}

	issues []*api.Issue
}

type Result struct {
	Tasks  []*api.Task
	Issues []*api.Issue

	// A helper field that lets some external post-processor
	// to inject new tasks correctly (e.g. Dockerfile build tasks)
	TasksCountBeforeFiltering int64
}

func New(opts ...Option) *Parser {
	parser := &Parser{
		environment:                 make(map[string]string),
		fs:                          dummy.New(),
		disabledTaskNamesAndAliases: make(map[string]struct{}),
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
	taskParser := task.NewTask(nil, nil, parser.additionalInstances, parser.additionalTaskProperties,
		parser.missingInstancesAllowed)
	pipeParser := task.NewDockerPipe(nil, nil, parser.additionalTaskProperties)
	builderParser := task.NewDockerBuilder(nil, nil, parser.additionalTaskProperties)
	parser.parsers = map[nameable.Nameable]parseable.Parseable{
		nameable.NewRegexNameable("^(.*)task$"):           taskParser,
		nameable.NewRegexNameable("^(.*)pipe$"):           pipeParser,
		nameable.NewRegexNameable("^(.*)docker_builder$"): builderParser,
	}

	return parser
}

func (p *Parser) registerIssuef(level api.Issue_Level, line int, column int, format string, args ...interface{}) {
	p.issues = append(p.issues, &api.Issue{
		Level:   level,
		Message: fmt.Sprintf(format, args...),
		Path:    pathYAML,
		Line:    uint64(line),
		Column:  uint64(column),
	})
}

// nolint:gocognit // it's a parser, there's a lot of logic going on
func (p *Parser) parseTasks(tree *node.Node) ([]task.ParseableTaskLike, error) {
	var tasks []task.ParseableTaskLike

	for _, treeItem := range tree.Children {
		if strings.HasPrefix(treeItem.Name, "task_") {
			p.registerIssuef(api.Issue_WARNING, treeItem.Line, treeItem.Column,
				"you've probably meant %s_task", strings.TrimPrefix(treeItem.Name, "task_"))
		}

		for key, value := range p.parsers {
			var taskLike task.ParseableTaskLike
			switch value.(type) {
			case *task.Task:
				taskLike = task.NewTask(
					environment.Copy(p.environment),
					p.boolevator,
					p.additionalInstances,
					p.additionalTaskProperties,
					p.missingInstancesAllowed,
				)
			case *task.DockerPipe:
				taskLike = task.NewDockerPipe(environment.Copy(p.environment), p.boolevator, p.additionalTaskProperties)
			case *task.DockerBuilder:
				taskLike = task.NewDockerBuilder(environment.Copy(p.environment), p.boolevator, p.additionalTaskProperties)
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

			taskLike.SetID(p.NextTaskID())

			// Emit a warning if the user tries to give multiple names to a single task
			var quickTaskName string

			if rn, ok := key.(*nameable.RegexNameable); ok {
				quickTaskName = rn.FirstGroupOrDefault(treeItem.Name, "")
			}

			if taskLike.Name() != "" && quickTaskName != "" {
				p.registerIssuef(api.Issue_WARNING, treeItem.Line, treeItem.Column,
					"giving a task multiple names (%q and %q) can be ambiguous",
					quickTaskName, taskLike.Name())
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

			p.tasksCountBeforeFiltering++

			enabled, err := taskLike.Enabled(environment.Merge(taskSpecificEnv, p.environment), p.boolevator)
			if err != nil {
				return nil, err
			}

			if !enabled {
				p.disabledTaskNamesAndAliases[taskLike.Name()] = struct{}{}
				p.disabledTaskNamesAndAliases[taskLike.Alias()] = struct{}{}
				continue
			}

			taskLike.SetIndexWithinBuild(p.NextTaskLocalIndex())

			tasks = append(tasks, taskLike)
		}
	}

	return tasks, nil
}

func (p *Parser) Parse(ctx context.Context, config string) (result *Result, err error) {
	defer func() {
		if re, ok := err.(*parsererror.Rich); ok {
			re.Enrich(config)
		}
	}()

	// Convert the parsed and nested YAML structure into a tree
	// to get the ability to walk parents
	tree, err := node.NewFromText(config)
	if err != nil {
		return nil, err
	}

	// Run modifiers on it
	if err := matrix.ExpandMatrices(tree); err != nil {
		return nil, err
	}

	// Run parsers on the top-level nodes
	tasks, err := p.parseTasks(tree)
	if err != nil {
		return nil, err
	}

	if err := p.resolveDependenciesShallow(tasks); err != nil {
		return nil, err
	}

	if len(tasks) == 0 {
		return &Result{Issues: p.issues}, nil
	}

	if err := validateDependenciesDeep(tasks); err != nil {
		return nil, err
	}

	var protoTasks []*api.Task
	for _, task := range tasks {
		protoTask := task.Proto().(*api.Task)

		if err := validateTask(protoTask); err != nil {
			return nil, err
		}

		protoTasks = append(protoTasks, protoTask)
	}

	// Create service tasks
	serviceTasks, err := p.createServiceTasks(ctx, protoTasks)
	if err != nil {
		return nil, err
	}
	protoTasks = append(protoTasks, serviceTasks...)

	// Postprocess individual tasks
	for _, protoTask := range protoTasks {
		// Insert empty clone instruction if custom clone script wasn't provided by the user
		ensureCloneInstruction(protoTask)

		// Provide unique labels for identically named tasks
		uniqueLabelsForTask, err := p.uniqueLabels(protoTask, protoTasks, p.additionalInstances)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", parsererror.ErrInternal, err)
		}
		protoTask.Metadata.UniqueLabels = uniqueLabelsForTask
	}

	// Sort tasks by their IDs to make output consistent across runs
	sort.Slice(protoTasks, func(i, j int) bool {
		return protoTasks[i].LocalGroupId < protoTasks[j].LocalGroupId
	})

	return &Result{
		Tasks:                     protoTasks,
		TasksCountBeforeFiltering: p.tasksCountBeforeFiltering,
		Issues:                    p.issues,
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
		p.idNumbering++
	}()
	return p.idNumbering
}

func (p *Parser) NextTaskLocalIndex() int64 {
	defer func() {
		p.indexNumbering++
	}()
	return p.indexNumbering
}

func (p *Parser) Schema() *schema.Schema {
	schema := &schema.Schema{
		Type:                 schema.PrimitiveTypes{schema.ObjectType},
		ID:                   "https://cirrus-ci.org/",
		Title:                "JSON schema for Cirrus CI configuration files",
		SchemaRef:            "http://json-schema.org/draft-04/schema#",
		Properties:           make(map[string]*schema.Schema),
		PatternProperties:    make(map[*regexp.Regexp]*schema.Schema),
		AdditionalProperties: &schema.AdditionalProperties{Schema: nil},
	}

	for parserName, parser := range p.parsers {
		switch nameable := parserName.(type) {
		case *nameable.SimpleNameable:
			schema.Properties[nameable.Name()] = parser.Schema()
		case *nameable.RegexNameable:
			schema.PatternProperties[nameable.Regex()] = parser.Schema()
		}

		// Note: this is a simplification that doesn't return collectible fields recursively,
		// because it assumes that we're only defining collectibles on the first depth level.
		for _, collectibleFields := range parser.CollectibleFields() {
			schema.Properties[collectibleFields.Name] = collectibleFields.Schema
		}
	}

	return schema
}

func (p *Parser) resolveDependenciesShallow(tasks []task.ParseableTaskLike) error {
	for _, task := range tasks {
		var dependsOnIDs []int64
		for _, dependsOnName := range task.DependsOnNames() {
			// Dependency may be missing due to only_if
			if _, ok := p.disabledTaskNamesAndAliases[dependsOnName]; ok {
				continue
			}

			var dependencyWasFound bool

			for _, subTask := range tasks {
				if subTask.Name() == dependsOnName || subTask.Alias() == dependsOnName {
					dependsOnIDs = append(dependsOnIDs, subTask.ID())
					dependencyWasFound = true
				}
			}

			if !dependencyWasFound {
				return fmt.Errorf("%w: there's no task '%s', but task '%s' depends on it",
					parsererror.ErrBasic, dependsOnName, task.Name())
			}
		}
		sort.Slice(dependsOnIDs, func(i, j int) bool { return dependsOnIDs[i] < dependsOnIDs[j] })
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
		Repository: fmt.Sprintf("cirrus-ci-community/%s", dockerfileHash),
		Reference:  "latest",
		Platform:   taskContainer.Platform,
		Dockerfile: taskContainer.Dockerfile,
		Arguments:  taskContainer.DockerArguments,
	}

	anyInstance, err := anypb.New(prebuiltInstance)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", parsererror.ErrInternal, err)
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

	script := fmt.Sprintf("docker build "+
		"--tag gcr.io/%s:%s "+
		"--file %s%s ",
		prebuiltInstance.Repository, prebuiltInstance.Reference,
		taskContainer.Dockerfile, dockerBuildArgs)

	if taskContainer.Platform == api.Platform_WINDOWS {
		script += "."
	} else {
		script += "${CIRRUS_DOCKER_CONTEXT:-$CIRRUS_WORKING_DIR}"
	}

	serviceTask := &api.Task{
		Name:         fmt.Sprintf("Prebuild %s%s", taskContainer.Dockerfile, buildArgs),
		LocalGroupId: p.NextTaskID(),
		Instance:     anyInstance,
		Commands: []*api.Command{
			{
				Name: "build",
				Instruction: &api.Command_ScriptInstruction{
					ScriptInstruction: &api.ScriptInstruction{
						Scripts: []string{script},
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
			Properties: environment.Merge(
				task.DefaultTaskProperties(),
				map[string]string{
					"skip_notifications": "true",
				},
			),
		},
	}

	// Some metadata property fields duplicate other fields
	serviceTask.Metadata.Properties["indexWithinBuild"] = strconv.FormatInt(p.NextTaskLocalIndex(), 10)

	// Some metadata property fields are preserved from the original task
	serviceTask.Metadata.Properties["timeout_in"] = protoTask.Metadata.Properties["timeout_in"]

	return serviceTask, nil
}

func (p *Parser) createServiceTasks(ctx context.Context, protoTasks []*api.Task) ([]*api.Task, error) {
	serviceTasks := make(map[string]*api.Task)

	for _, protoTask := range protoTasks {
		if protoTask.Instance == nil && p.missingInstancesAllowed {
			continue
		}

		dynamicInstance, err := anypb.UnmarshalNew(protoTask.Instance, proto.UnmarshalOptions{})

		if errors.Is(err, protoregistry.NotFound) {
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("%w: failed to unmarshal task's instance: %v", parsererror.ErrInternal, err)
		}

		taskContainer, ok := dynamicInstance.(*api.ContainerInstance)
		if !ok {
			continue
		}

		if taskContainer.Dockerfile == "" {
			continue
		}

		// Craft Docker build arguments: arguments used in content hash calculation
		var hashableArgsSlice []string
		for key, value := range taskContainer.DockerArguments {
			hashableArgsSlice = append(hashableArgsSlice, key+value)
		}
		sort.Strings(hashableArgsSlice)
		hashableArgs := strings.Join(hashableArgsSlice, ", ")

		dockerfileHash, err := p.fileHash(ctx, taskContainer.Dockerfile, []byte(hashableArgs))
		if err != nil {
			return nil, err
		}

		// Find or create service task
		serviceTaskKey := taskContainer.Dockerfile + hashableArgs

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
		updatedInstance, err := anypb.New(taskContainer)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", parsererror.ErrInternal, err)
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
