package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hiroki-koketsu/go-otel-sample/internal/model"
	"github.com/hiroki-koketsu/go-otel-sample/internal/repository"
	"github.com/hiroki-koketsu/go-otel-sample/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("github.com/hiroki-koketsu/go-otel-sample/internal/handler")

// TaskHandler handles HTTP requests for tasks.
type TaskHandler struct {
	repo    *repository.TaskRepository
	logger  *slog.Logger
	metrics *telemetry.Metrics
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(repo *repository.TaskRepository, logger *slog.Logger, metrics *telemetry.Metrics) *TaskHandler {
	return &TaskHandler{
		repo:    repo,
		logger:  logger,
		metrics: metrics,
	}
}

// Routes returns the chi router with task routes.
func (h *TaskHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
}

// List returns all tasks.
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	ctx, span := tracer.Start(ctx, "TaskHandler.List")
	defer span.End()

	h.logger.InfoContext(ctx, "listing all tasks")

	tasks, err := h.repo.List(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list tasks", slog.Any("error", err))
		h.respondError(w, http.StatusInternalServerError, "failed to list tasks")
		h.recordMetrics(ctx, "GET", "/api/v1/tasks", http.StatusInternalServerError, start)
		return
	}

	span.SetAttributes(attribute.Int("task.count", len(tasks)))
	h.logger.InfoContext(ctx, "tasks listed", slog.Int("count", len(tasks)))

	h.respondJSON(w, http.StatusOK, tasks)
	h.recordMetrics(ctx, "GET", "/api/v1/tasks", http.StatusOK, start)
}

// Create adds a new task.
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	ctx, span := tracer.Start(ctx, "TaskHandler.Create")
	defer span.End()

	var req model.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WarnContext(ctx, "invalid request body", slog.Any("error", err))
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		h.recordMetrics(ctx, "POST", "/api/v1/tasks", http.StatusBadRequest, start)
		return
	}

	if err := req.Validate(); err != nil {
		h.logger.WarnContext(ctx, "validation failed", slog.Any("error", err))
		h.respondError(w, http.StatusBadRequest, err.Error())
		h.recordMetrics(ctx, "POST", "/api/v1/tasks", http.StatusBadRequest, start)
		return
	}

	h.logger.InfoContext(ctx, "creating task", slog.String("title", req.Title))

	task, err := h.repo.Create(ctx, &req)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to create task", slog.Any("error", err))
		h.respondError(w, http.StatusInternalServerError, "failed to create task")
		h.recordMetrics(ctx, "POST", "/api/v1/tasks", http.StatusInternalServerError, start)
		return
	}

	span.SetAttributes(attribute.String("task.id", task.ID))
	h.logger.InfoContext(ctx, "task created", slog.String("id", task.ID))

	h.respondJSON(w, http.StatusCreated, task)
	h.recordMetrics(ctx, "POST", "/api/v1/tasks", http.StatusCreated, start)
}

// GetByID returns a task by ID.
func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()
	id := chi.URLParam(r, "id")

	ctx, span := tracer.Start(ctx, "TaskHandler.GetByID",
		trace.WithAttributes(attribute.String("task.id", id)),
	)
	defer span.End()

	h.logger.InfoContext(ctx, "getting task", slog.String("id", id))

	task, err := h.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, model.ErrTaskNotFound) {
			h.logger.WarnContext(ctx, "task not found", slog.String("id", id))
			h.respondError(w, http.StatusNotFound, "task not found")
			h.recordMetrics(ctx, "GET", "/api/v1/tasks/{id}", http.StatusNotFound, start)
			return
		}
		h.logger.ErrorContext(ctx, "failed to get task", slog.Any("error", err))
		h.respondError(w, http.StatusInternalServerError, "failed to get task")
		h.recordMetrics(ctx, "GET", "/api/v1/tasks/{id}", http.StatusInternalServerError, start)
		return
	}

	h.logger.InfoContext(ctx, "task retrieved", slog.String("id", id))

	h.respondJSON(w, http.StatusOK, task)
	h.recordMetrics(ctx, "GET", "/api/v1/tasks/{id}", http.StatusOK, start)
}

// Update modifies an existing task.
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()
	id := chi.URLParam(r, "id")

	ctx, span := tracer.Start(ctx, "TaskHandler.Update",
		trace.WithAttributes(attribute.String("task.id", id)),
	)
	defer span.End()

	var req model.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WarnContext(ctx, "invalid request body", slog.Any("error", err))
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		h.recordMetrics(ctx, "PUT", "/api/v1/tasks/{id}", http.StatusBadRequest, start)
		return
	}

	h.logger.InfoContext(ctx, "updating task", slog.String("id", id))

	task, err := h.repo.Update(ctx, id, &req)
	if err != nil {
		if errors.Is(err, model.ErrTaskNotFound) {
			h.logger.WarnContext(ctx, "task not found", slog.String("id", id))
			h.respondError(w, http.StatusNotFound, "task not found")
			h.recordMetrics(ctx, "PUT", "/api/v1/tasks/{id}", http.StatusNotFound, start)
			return
		}
		h.logger.ErrorContext(ctx, "failed to update task", slog.Any("error", err))
		h.respondError(w, http.StatusInternalServerError, "failed to update task")
		h.recordMetrics(ctx, "PUT", "/api/v1/tasks/{id}", http.StatusInternalServerError, start)
		return
	}

	h.logger.InfoContext(ctx, "task updated", slog.String("id", id))

	h.respondJSON(w, http.StatusOK, task)
	h.recordMetrics(ctx, "PUT", "/api/v1/tasks/{id}", http.StatusOK, start)
}

// Delete removes a task.
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()
	id := chi.URLParam(r, "id")

	ctx, span := tracer.Start(ctx, "TaskHandler.Delete",
		trace.WithAttributes(attribute.String("task.id", id)),
	)
	defer span.End()

	h.logger.InfoContext(ctx, "deleting task", slog.String("id", id))

	err := h.repo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, model.ErrTaskNotFound) {
			h.logger.WarnContext(ctx, "task not found", slog.String("id", id))
			h.respondError(w, http.StatusNotFound, "task not found")
			h.recordMetrics(ctx, "DELETE", "/api/v1/tasks/{id}", http.StatusNotFound, start)
			return
		}
		h.logger.ErrorContext(ctx, "failed to delete task", slog.Any("error", err))
		h.respondError(w, http.StatusInternalServerError, "failed to delete task")
		h.recordMetrics(ctx, "DELETE", "/api/v1/tasks/{id}", http.StatusInternalServerError, start)
		return
	}

	h.logger.InfoContext(ctx, "task deleted", slog.String("id", id))

	w.WriteHeader(http.StatusNoContent)
	h.recordMetrics(ctx, "DELETE", "/api/v1/tasks/{id}", http.StatusNoContent, start)
}

// Health returns a health check response.
func (h *TaskHandler) Health(w http.ResponseWriter, r *http.Request) {
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *TaskHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func (h *TaskHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

func (h *TaskHandler) recordMetrics(ctx context.Context, method, route string, status int, start time.Time) {
	duration := time.Since(start).Seconds()

	attrs := metric.WithAttributes(
		attribute.String("http.method", method),
		attribute.String("http.route", route),
		attribute.Int("http.status_code", status),
	)

	h.metrics.RequestCounter.Add(ctx, 1, attrs)
	h.metrics.RequestDuration.Record(ctx, duration, attrs)
}
