package parser

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/environment"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/cachinglayer"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/dummy"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/abstractcontainer"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/issue"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/modifier/matrix"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parserkit"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/task"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/lestrrat-go/jsschema"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	metadataPropertyDockerfileHash = "dockerfile_hash"
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

	parserKit *parserkit.ParserKit

	parsers                  map[nameable.Nameable]parseable.Parseable
	idNumbering              int64
	indexNumbering           int64
	additionalInstances      map[string]protoreflect.MessageDescriptor
	additionalTaskProperties []*descriptor.FieldDescriptorProto
	missingInstancesAllowed  bool

	tasksCountBeforeFiltering   int64
	disabledTaskNamesAndAliases map[string]struct{}
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

	// Wrap the final file system in a caching layer
	wrappedFS, err := cachinglayer.Wrap(parser.fs)
	if err != nil {
		panic(err)
	}
	parser.fs = wrappedFS

	// Initialize boolevator
	parser.parserKit = &parserkit.ParserKit{
		Boolevator: boolevator.New(boolevator.WithFunctions(map[string]boolevator.Function{
			"changesInclude":     parser.bfuncChangesInclude(),
			"changesIncludeOnly": parser.bfuncChangesIncludeOnly(),
		})),
		IssueRegistry: issue.NewRegistry(),
	}

	// Register parsers
	taskParser := task.NewTask(nil, nil, parser.additionalInstances, parser.additionalTaskProperties,
		parser.missingInstancesAllowed, 0, 0)
	pipeParser := task.NewDockerPipe(nil, nil, parser.additionalTaskProperties, 0, 0)
	builderParser := task.NewDockerBuilder(nil, nil, parser.additionalTaskProperties, 0, 0)
	parser.parsers = map[nameable.Nameable]parseable.Parseable{
		nameable.NewRegexNameable("^(.*)task$"):           taskParser,
		nameable.NewRegexNameable("^(.*)pipe$"):           pipeParser,
		nameable.NewRegexNameable("^(.*)docker_builder$"): builderParser,
	}

	return parser
}

func (p *Parser) parseTasks(tree *node.Node) ([]task.ParseableTaskLike, error) {
	var tasks []task.ParseableTaskLike

	for _, treeItem := range tree.Children {
		if strings.HasPrefix(treeItem.Name, "task_") {
			p.parserKit.IssueRegistry.RegisterIssuef(api.Issue_WARNING, treeItem.Line, treeItem.Column,
				"you've probably meant %s_task", strings.TrimPrefix(treeItem.Name, "task_"))
		}

		for key, value := range p.parsers {
			var taskLike task.ParseableTaskLike
			switch value.(type) {
			case *task.Task:
				taskLike = task.NewTask(
					environment.Copy(p.environment),
					p.parserKit,
					p.additionalInstances,
					p.additionalTaskProperties,
					p.missingInstancesAllowed,
					treeItem.Line,
					treeItem.Column,
				)
			case *task.DockerPipe:
				taskLike = task.NewDockerPipe(
					environment.Copy(p.environment),
					p.parserKit,
					p.additionalTaskProperties,
					treeItem.Line,
					treeItem.Column,
				)
			case *task.DockerBuilder:
				taskLike = task.NewDockerBuilder(
					environment.Copy(p.environment),
					p.parserKit,
					p.additionalTaskProperties,
					treeItem.Line,
					treeItem.Column,
				)
			default:
				panic("unknown task-like object")
			}

			if !key.Matches(treeItem.Name) {
				continue
			}

			err := taskLike.Parse(treeItem, p.parserKit)
			if err != nil {
				return nil, err
			}

			taskLike.SetID(p.NextTaskID())

			// Set task's name if not set in the definition
			if rn, ok := key.(*nameable.RegexNameable); ok {
				taskLike.SetFallbackName(rn.FirstGroupOrDefault(treeItem.Name, "main"))
			}

			if taskLike.Name() == "" {
				taskLike.SetName(taskLike.FallbackName())
			}

			// Filtering based on "only_if" expression evaluation
			taskSpecificEnv := map[string]string{
				"CIRRUS_TASK_NAME": taskLike.Name(),
			}

			p.tasksCountBeforeFiltering++

			enabled, err := taskLike.Enabled(environment.Merge(taskSpecificEnv, p.environment), p.parserKit.Boolevator)
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

//nolint:gocognit // it's a parser, and it's complicated
func (p *Parser) Parse(ctx context.Context, config string) (result *Result, err error) {
	defer func() {
		if re, ok := err.(*parsererror.Rich); ok {
			re.Enrich(config)
		}
	}()

	// Register additional instances
	for _, additionalInstance := range p.additionalInstances {
		_, err := protoregistry.GlobalTypes.FindMessageByName(additionalInstance.FullName())
		if err == nil {
			continue
		} else if !errors.Is(err, protoregistry.NotFound) {
			return nil, err
		}

		additionalType := dynamicpb.NewMessageType(additionalInstance)
		if err := protoregistry.GlobalTypes.RegisterMessage(additionalType); err != nil {
			return nil, err
		}
	}

	// Work around Cirus Cloud parser's historically lax merging
	// of the YAML aliases[1]. See yaml-merge-sequence.yml and
	// yaml-merge-collectible.yml for examples of conflicting
	// behaviors that we should support.
	//
	// [1]: https://yaml.org/type/merge.html
	var mergeExemptions []nameable.Nameable

	for _, parser := range p.parsers {
		for _, field := range parser.CollectibleFields() {
			mergeExemptions = append(mergeExemptions, nameable.NewSimpleNameable(field.Name))
		}

		for _, field := range parser.Fields() {
			if !field.Repeatable() {
				continue
			}

			mergeExemptions = append(mergeExemptions, field.Name())
		}
	}

	// Convert the parsed and nested YAML structure into a tree
	// to get the ability to walk parents
	tree, err := node.NewFromTextWithMergeExemptions(config, mergeExemptions)
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
		return &Result{Issues: p.parserKit.IssueRegistry.Issues()}, nil
	}

	if err := validateDependenciesDeep(tasks); err != nil {
		return nil, err
	}

	p.searchForUnbalancedOnlyIfs(tasks)

	protoTaskToInstanceNode := map[int64]*node.Node{}
	var protoTasks []*api.Task
	for _, task := range tasks {
		type HasInstanceNode interface {
			InstanceNode() *node.Node
		}
		if taskWithInstanceNode, ok := task.(HasInstanceNode); ok {
			protoTaskToInstanceNode[task.ID()] = taskWithInstanceNode.InstanceNode()
		}

		protoTask := task.Proto().(*api.Task)

		if err := validateTask(protoTask); err != nil {
			return nil, err
		}

		protoTasks = append(protoTasks, protoTask)
	}

	// Calculate Dockerfile hashes that will be used
	// to create service tasks and in the Cirrus Cloud
	if err := p.calculateDockerfileHashes(ctx, protoTasks, protoTaskToInstanceNode); err != nil {
		return nil, err
	}

	// Create service tasks
	serviceTasks, err := p.createServiceTasks(protoTasks)
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
		Issues:                    p.parserKit.IssueRegistry.Issues(),
	}, nil
}

func (p *Parser) searchForUnbalancedOnlyIfs(tasks []task.ParseableTaskLike) {
	// Create an index
	idx := map[int64]task.ParseableTaskLike{}

	for _, task := range tasks {
		idx[task.ID()] = task
	}

	// Analyze dependencies
	for _, task := range tasks {
		for _, dependsOnID := range task.DependsOnIDs() {
			dependent, ok := idx[dependsOnID]
			if !ok {
				continue
			}

			if dependent.OnlyIfExpression() != "" && task.OnlyIfExpression() != dependent.OnlyIfExpression() {
				p.registerUnbalancedOnlyIfIssue(task, dependent.Name())
			}
		}
	}
}

func (p *Parser) registerUnbalancedOnlyIfIssue(dependent task.ParseableTaskLike, dependeeName string) {
	p.parserKit.IssueRegistry.RegisterIssuef(api.Issue_WARNING, dependent.Line(), dependent.Column(),
		"task \"%s\" depends on task \"%s\", but their only_if conditions are different",
		dependent.Name(), dependeeName)
}

func (p *Parser) ParseFromFile(ctx context.Context, path string) (*Result, error) {
	config, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return p.Parse(ctx, string(config))
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
	fallbackNames := make(map[string]struct{})

	for _, task := range tasks {
		fallbackNames[task.FallbackName()] = struct{}{}
	}

	for _, task := range tasks {
		var dependsOnIDs []int64
		for _, dependsOnName := range task.DependsOnNames() {
			// Dependency may be missing due to only_if
			if _, ok := p.disabledTaskNamesAndAliases[dependsOnName]; ok {
				p.registerUnbalancedOnlyIfIssue(task, dependsOnName)
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
				if _, ok := fallbackNames[dependsOnName]; ok {
					return fmt.Errorf("%w: task '%s' depends on task '%s', which name was overridden by "+
						"a name: field inside of that task", parsererror.ErrBasic, task.Name(), dependsOnName)
				}

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
	abstractContainer abstractcontainer.AbstractContainer,
) (*api.Task, error) {
	prebuiltInstance := &api.PrebuiltImageInstance{
		Repository: fmt.Sprintf("cirrus-ci-community/%s", dockerfileHash),
		Reference:  "latest",
		Platform:   abstractContainer.Platform(),
		Dockerfile: abstractContainer.Dockerfile(),
		Arguments:  abstractContainer.DockerArguments(),
	}

	anyInstance, err := anypb.New(prebuiltInstance)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", parsererror.ErrInternal, err)
	}

	// Craft Docker build arguments: task name
	var buildArgsSlice []string
	for key, value := range abstractContainer.DockerArguments() {
		buildArgsSlice = append(buildArgsSlice, fmt.Sprintf("%s=%s", key, value))
	}
	sort.Strings(buildArgsSlice)
	var buildArgs string
	for _, buildArg := range buildArgsSlice {
		buildArgs += fmt.Sprintf(" %s", buildArg)
	}

	// Craft Docker build arguments: docker build command
	var dockerBuildArgsSlice []string
	for key, value := range abstractContainer.DockerArguments() {
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
		abstractContainer.Dockerfile(), dockerBuildArgs)

	if abstractContainer.Platform() == api.Platform_WINDOWS {
		script += "."
	} else {
		script += "${CIRRUS_DOCKER_CONTEXT:-$CIRRUS_WORKING_DIR}"
	}

	serviceTask := &api.Task{
		Name:         fmt.Sprintf("Prebuild %s%s", abstractContainer.Dockerfile(), buildArgs),
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

func (p *Parser) createServiceTasks(protoTasks []*api.Task) ([]*api.Task, error) {
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

		var abstractContainer abstractcontainer.AbstractContainer

		switch instance := dynamicInstance.(type) {
		case *api.ContainerInstance:
			abstractContainer = &abstractcontainer.ContainerInstance{
				Proto: instance,
			}
		case *api.PersistentWorkerInstance:
			if instance.Isolation == nil {
				continue
			}

			container := instance.Isolation.GetContainer()
			if container == nil {
				continue
			}

			abstractContainer = &abstractcontainer.IsolationContainer{
				Proto: instance,
			}
		default:
			continue
		}

		if abstractContainer.Dockerfile() == "" {
			continue
		}

		// Retrieve the Dockerfile hash calculated for this task earlier in the parsing routine
		dockerfileHash, ok := protoTask.Metadata.Properties[metadataPropertyDockerfileHash]
		if !ok {
			return nil, fmt.Errorf("%w: %q is missing it's Dockerfile hash which should've been pre-calculated",
				parsererror.ErrInternal, protoTask.Name)
		}

		// Find or create service task
		serviceTask, ok := serviceTasks[dockerfileHash]
		if !ok {
			serviceTask, err = p.createServiceTask(dockerfileHash, protoTask, abstractContainer)
			if err != nil {
				return nil, err
			}

			serviceTasks[dockerfileHash] = serviceTask
		}

		// Set dependency to the found or created service task
		protoTask.RequiredGroups = append(protoTask.RequiredGroups, serviceTask.LocalGroupId)

		// Ensure that the task will use our to-be-created image
		abstractContainer.SetImage(fmt.Sprintf("gcr.io/cirrus-ci-community/%s:latest", dockerfileHash))
		updatedInstance, err := anypb.New(abstractContainer.Message())
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
