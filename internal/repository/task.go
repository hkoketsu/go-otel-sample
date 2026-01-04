package repository

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hiroki-koketsu/go-otel-sample/internal/model"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("github.com/hiroki-koketsu/go-otel-sample/internal/repository")

// TaskRepository provides an in-memory storage for tasks.
type TaskRepository struct {
	mu    sync.RWMutex
	tasks map[string]*model.Task
}

// NewTaskRepository creates a new TaskRepository.
func NewTaskRepository() *TaskRepository {
	return &TaskRepository{
		tasks: make(map[string]*model.Task),
	}
}

// Create adds a new task to the repository.
func (r *TaskRepository) Create(ctx context.Context, req *model.CreateTaskRequest) (*model.Task, error) {
	ctx, span := tracer.Start(ctx, "TaskRepository.Create",
		trace.WithAttributes(attribute.String("task.title", req.Title)),
	)
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	task := &model.Task{
		ID:          uuid.New().String(),
		Title:       req.Title,
		Description: req.Description,
		Done:        false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	r.tasks[task.ID] = task

	span.SetAttributes(attribute.String("task.id", task.ID))
	return task, nil
}

// GetByID retrieves a task by its ID.
func (r *TaskRepository) GetByID(ctx context.Context, id string) (*model.Task, error) {
	ctx, span := tracer.Start(ctx, "TaskRepository.GetByID",
		trace.WithAttributes(attribute.String("task.id", id)),
	)
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	task, ok := r.tasks[id]
	if !ok {
		span.SetAttributes(attribute.Bool("task.found", false))
		return nil, model.ErrTaskNotFound
	}

	span.SetAttributes(attribute.Bool("task.found", true))
	return task, nil
}

// List returns all tasks in the repository.
func (r *TaskRepository) List(ctx context.Context) ([]*model.Task, error) {
	ctx, span := tracer.Start(ctx, "TaskRepository.List")
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]*model.Task, 0, len(r.tasks))
	for _, task := range r.tasks {
		tasks = append(tasks, task)
	}

	span.SetAttributes(attribute.Int("task.count", len(tasks)))
	return tasks, nil
}

// Update modifies an existing task.
func (r *TaskRepository) Update(ctx context.Context, id string, req *model.UpdateTaskRequest) (*model.Task, error) {
	ctx, span := tracer.Start(ctx, "TaskRepository.Update",
		trace.WithAttributes(attribute.String("task.id", id)),
	)
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		span.SetAttributes(attribute.Bool("task.found", false))
		return nil, model.ErrTaskNotFound
	}

	if req.Title != "" {
		task.Title = req.Title
	}
	if req.Description != "" {
		task.Description = req.Description
	}
	if req.Done != nil {
		task.Done = *req.Done
	}
	task.UpdatedAt = time.Now()

	span.SetAttributes(attribute.Bool("task.found", true))
	return task, nil
}

// Delete removes a task from the repository.
func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "TaskRepository.Delete",
		trace.WithAttributes(attribute.String("task.id", id)),
	)
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tasks[id]; !ok {
		span.SetAttributes(attribute.Bool("task.found", false))
		return model.ErrTaskNotFound
	}

	delete(r.tasks, id)
	span.SetAttributes(attribute.Bool("task.found", true))
	return nil
}

// Count returns the current number of tasks.
func (r *TaskRepository) Count() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.tasks))
}
