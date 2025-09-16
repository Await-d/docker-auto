package scheduler

import (
	"fmt"
	"sync"

	"docker-auto/internal/model"
)

// DefaultTaskRegistry implements the TaskRegistry interface
type DefaultTaskRegistry struct {
	tasks map[model.TaskType]TaskFactory
	mu    sync.RWMutex
}

// NewTaskRegistry creates a new task registry
func NewTaskRegistry() TaskRegistry {
	return &DefaultTaskRegistry{
		tasks: make(map[model.TaskType]TaskFactory),
	}
}

// RegisterTask registers a new task type with its factory
func (r *DefaultTaskRegistry) RegisterTask(taskType model.TaskType, factory TaskFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[taskType]; exists {
		return fmt.Errorf("task type %s is already registered", taskType)
	}

	r.tasks[taskType] = factory
	return nil
}

// GetTask creates a task instance by type
func (r *DefaultTaskRegistry) GetTask(taskType model.TaskType) (Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.tasks[taskType]
	if !exists {
		return nil, fmt.Errorf("task type %s is not registered", taskType)
	}

	return factory(), nil
}

// GetRegisteredTypes returns all registered task types
func (r *DefaultTaskRegistry) GetRegisteredTypes() []model.TaskType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]model.TaskType, 0, len(r.tasks))
	for taskType := range r.tasks {
		types = append(types, taskType)
	}

	return types
}

// UnregisterTask removes a task type from registry
func (r *DefaultTaskRegistry) UnregisterTask(taskType model.TaskType) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[taskType]; !exists {
		return fmt.Errorf("task type %s is not registered", taskType)
	}

	delete(r.tasks, taskType)
	return nil
}