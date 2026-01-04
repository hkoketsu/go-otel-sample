package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiroki-koketsu/go-otel-sample/internal/config"
	"github.com/hiroki-koketsu/go-otel-sample/internal/handler"
	"github.com/hiroki-koketsu/go-otel-sample/internal/repository"
	"github.com/hiroki-koketsu/go-otel-sample/internal/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Create a basic logger for startup (before OTel is initialized)
	startupLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	startupLogger.Info("starting application",
		slog.String("service", cfg.ServiceName),
		slog.String("environment", cfg.Environment),
		slog.String("port", cfg.ServerPort),
	)

	ctx := context.Background()

	// Initialize OpenTelemetry tracer provider
	tp, err := telemetry.InitTracerProvider(ctx, cfg.ServiceName, cfg.OTLPEndpoint, cfg.Environment)
	if err != nil {
		startupLogger.Error("failed to initialize tracer provider", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			startupLogger.Error("failed to shutdown tracer provider", slog.Any("error", err))
		}
	}()

	// Initialize OpenTelemetry meter provider
	mp, err := telemetry.InitMeterProvider(ctx, cfg.ServiceName, cfg.OTLPEndpoint, cfg.Environment)
	if err != nil {
		startupLogger.Error("failed to initialize meter provider", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := mp.Shutdown(ctx); err != nil {
			startupLogger.Error("failed to shutdown meter provider", slog.Any("error", err))
		}
	}()

	// Initialize task repository
	taskRepo := repository.NewTaskRepository()

	// Initialize OpenTelemetry logger provider (after other providers for log-trace correlation)
	lp, logger, err := telemetry.InitLoggerProvider(ctx, cfg.ServiceName, cfg.OTLPEndpoint, cfg.Environment)
	if err != nil {
		startupLogger.Error("failed to initialize logger provider", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := lp.Shutdown(ctx); err != nil {
			startupLogger.Error("failed to shutdown logger provider", slog.Any("error", err))
		}
	}()

	// Create metrics instruments
	meter := otel.Meter(cfg.ServiceName)
	metrics, err := telemetry.NewMetrics(meter, taskRepo.Count)
	if err != nil {
		logger.Error("failed to create metrics", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize handlers
	taskHandler := handler.NewTaskHandler(taskRepo, logger, metrics)

	// Create router
	r := chi.NewRouter()

	// Apply standard middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health check endpoint (excluded from tracing)
	r.Get("/health", taskHandler.Health)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Mount("/tasks", taskHandler.Routes())
	})

	// Wrap router with OpenTelemetry HTTP instrumentation
	otelHandler := otelhttp.NewHandler(r, "http-server",
		otelhttp.WithFilter(func(r *http.Request) bool {
			// Skip tracing for health checks
			return r.URL.Path != "/health"
		}),
	)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      otelHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("server listening", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Create context with timeout for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Gracefully shutdown the server
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", slog.Any("error", err))
	}

	logger.Info("server stopped")
}
