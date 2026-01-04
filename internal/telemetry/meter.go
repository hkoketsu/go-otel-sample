package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Metrics holds the custom metrics instruments for the application.
type Metrics struct {
	RequestCounter   metric.Int64Counter
	RequestDuration  metric.Float64Histogram
	TasksGauge       metric.Int64ObservableGauge
	taskCountFunc    func() int64
}

// InitMeterProvider initializes the OpenTelemetry meter provider.
// It configures an OTLP gRPC exporter and sets up the global meter provider.
func InitMeterProvider(ctx context.Context, serviceName, otlpEndpoint, environment string) (*sdkmetric.MeterProvider, error) {
	// Create OTLP gRPC exporter
	conn, err := grpc.NewClient(otlpEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.DeploymentEnvironment(environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create meter provider with periodic reader (10 second interval)
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(10*time.Second),
		)),
		sdkmetric.WithResource(res),
	)

	// Set global meter provider
	otel.SetMeterProvider(mp)

	return mp, nil
}

// NewMetrics creates and registers custom metrics instruments.
func NewMetrics(meter metric.Meter, taskCountFunc func() int64) (*Metrics, error) {
	m := &Metrics{
		taskCountFunc: taskCountFunc,
	}

	var err error

	// Counter for total HTTP requests
	m.RequestCounter, err = meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request counter: %w", err)
	}

	// Histogram for request duration
	m.RequestDuration, err = meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request duration histogram: %w", err)
	}

	// Observable gauge for current task count
	m.TasksGauge, err = meter.Int64ObservableGauge(
		"tasks_total",
		metric.WithDescription("Current number of tasks in the system"),
		metric.WithUnit("{task}"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(m.taskCountFunc())
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tasks gauge: %w", err)
	}

	return m, nil
}
