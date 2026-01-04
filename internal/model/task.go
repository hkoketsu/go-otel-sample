package model

import (
	"time"
)

// Task represents a todo item in the system.
type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Done        bool      `json:"done"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateTaskRequest represents the request body for creating a task.
type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// UpdateTaskRequest represents the request body for updating a task.
type UpdateTaskRequest struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Done        *bool  `json:"done,omitempty"`
}

// Validate checks if the CreateTaskRequest is valid.
func (r *CreateTaskRequest) Validate() error {
	if r.Title == "" {
		return ErrTitleRequired
	}
	return nil
}

// TaskError represents a domain error for tasks.
type TaskError struct {
	Message string
}

func (e TaskError) Error() string {
	return e.Message
}

var (
	ErrTaskNotFound  = TaskError{Message: "task not found"}
	ErrTitleRequired = TaskError{Message: "title is required"}
)
