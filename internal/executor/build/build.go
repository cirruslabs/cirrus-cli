package build

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/build/taskstatus"
	"github.com/cirruslabs/cirrus-cli/internal/executor/cache"
	"github.com/cirruslabs/cirrus-cli/internal/logger"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"path/filepath"
)

type Build struct {
	// A directory on host where .cirrus.yml that drives this execution is located
	ProjectDir string

	Cache *cache.Cache

	// The actual tasks comprising this build
	tasks map[int64]*Task
}

func New(projectDir string, tasks []*api.Task, logger logger.Lightweight) (*Build, error) {
	// Normalize project directory path on host as it might be
	// simply ".", which is not suitable for bind mounting it
	// later to the container
	absoluteProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, err
	}

	// Wrap Protocol Buffers tasks
	wrappedTasks := make(map[int64]*Task)
	for _, task := range tasks {
		wrappedTask, err := NewFromProto(task, logger)
		if err != nil {
			return nil, err
		}
		wrappedTasks[wrappedTask.ID] = wrappedTask
	}

	c, err := cache.New("", filepath.Base(absoluteProjectDir))
	if err != nil {
		return nil, err
	}

	return &Build{
		ProjectDir: absoluteProjectDir,
		Cache:      c,
		tasks:      wrappedTasks,
	}, nil
}

func (b *Build) Tasks() (result []*Task) {
	for _, task := range b.tasks {
		result = append(result, task)
	}

	return
}

func (b *Build) GetTask(id int64) *Task {
	task, found := b.tasks[id]
	if !found {
		return nil
	}

	return task
}

func (b *Build) GetTaskFromIdentification(tid *api.TaskIdentification, clientSecret string) (*Task, error) {
	if tid.Secret != clientSecret {
		return nil, status.Error(codes.Unauthenticated, "provided secret value is invalid")
	}

	task, found := b.tasks[tid.TaskId]
	if !found {
		return nil, status.Errorf(codes.NotFound, "cannot find the task with the specified ID")
	}

	return task, nil
}

func (b *Build) taskHasUnresolvedDependencies(task *Task) bool {
	for _, requiredGroup := range task.RequiredIDs {
		requiredTask := b.GetTask(requiredGroup)

		if requiredTask.Status() == taskstatus.New {
			return true
		}
	}

	return false
}

func (b *Build) GetNextTask() (result *Task) {
	for _, task := range b.tasks {
		if task.Status() != taskstatus.New || b.taskHasUnresolvedDependencies(task) {
			continue
		}

		if result == nil || task.ID < result.ID {
			result = task
		}
	}

	return
}
