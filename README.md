# Go Samples - Learning Go with OpenTelemetry

A learning project for server-side Go development with full OpenTelemetry observability (traces, metrics, logs) deployed on local Kubernetes.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Docker Desktop K8s                          │
│                                                                     │
│  ┌──────────────┐      ┌───────────────────┐                       │
│  │              │      │                   │                       │
│  │  Go Samples  │─────▶│  OTel Collector   │                       │
│  │  (REST API)  │      │                   │                       │
│  │              │      └─────────┬─────────┘                       │
│  └──────────────┘                │                                 │
│                                  │                                 │
│           ┌──────────────────────┼──────────────────────┐          │
│           │                      │                      │          │
│           ▼                      ▼                      ▼          │
│    ┌──────────┐          ┌──────────────┐        ┌──────────┐     │
│    │  Jaeger  │          │  Prometheus  │        │   Loki   │     │
│    │ (Traces) │          │  (Metrics)   │        │  (Logs)  │     │
│    └────┬─────┘          └──────┬───────┘        └────┬─────┘     │
│         │                       │                     │           │
│         └───────────────────────┼─────────────────────┘           │
│                                 │                                 │
│                          ┌──────▼──────┐                          │
│                          │   Grafana   │                          │
│                          │ (Dashboard) │                          │
│                          └─────────────┘                          │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- [Go 1.22+](https://golang.org/dl/)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) with Kubernetes enabled
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [curl](https://curl.se/) and [jq](https://jqlang.github.io/jq/) (for API testing)

## Quick Start

### 1. Clone and Build

```bash
cd /path/to/go-otel-sample

# Download dependencies
go mod tidy

# Build the Docker image
make docker-build
```

### 2. Deploy Observability Stack

```bash
# Deploy Jaeger, Prometheus, Loki, Grafana, and OTel Collector
make k8s-observability

# Wait for pods to be ready
kubectl get pods -n go-otel-sample -w
```

### 3. Deploy Application

```bash
# Deploy the Go application
make k8s-deploy

# Check deployment status
kubectl get pods -n go-otel-sample
```

### 4. Access Services

```bash
# Port forward all services (run in separate terminal)
make port-forward
```

Access the UIs:
- **Grafana**: http://localhost:3000 (admin/admin)
- **Jaeger**: http://localhost:16686
- **Prometheus**: http://localhost:9090
- **Application**: http://localhost:8080

### 5. Test the API

```bash
# Quick API test
make api-test

# Or manually:
# Health check
curl http://localhost:8080/health

# Create a task
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Learn Go", "description": "Complete the tutorial"}'

# List tasks
curl http://localhost:8080/api/v1/tasks
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/tasks` | List all tasks |
| POST | `/api/v1/tasks` | Create a task |
| GET | `/api/v1/tasks/{id}` | Get task by ID |
| PUT | `/api/v1/tasks/{id}` | Update a task |
| DELETE | `/api/v1/tasks/{id}` | Delete a task |

### Example Requests

```bash
# Create task
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Buy groceries", "description": "Milk, bread, eggs"}'

# Update task (mark as done)
curl -X PUT http://localhost:8080/api/v1/tasks/{id} \
  -H "Content-Type: application/json" \
  -d '{"done": true}'

# Delete task
curl -X DELETE http://localhost:8080/api/v1/tasks/{id}
```

## Observability Features

### Traces (Jaeger)

The application automatically creates spans for:
- HTTP requests (via `otelhttp` middleware)
- Repository operations (Create, GetByID, List, Update, Delete)

View traces at http://localhost:16686:
1. Select "go-otel-sample" from the Service dropdown
2. Click "Find Traces"
3. Click on a trace to see the span waterfall

### Metrics (Prometheus)

Custom metrics exposed:
- `go_samples_http_requests_total` - Counter of HTTP requests
- `go_samples_http_request_duration_seconds` - Histogram of request durations
- `go_samples_tasks_total` - Gauge of current task count

Query metrics at http://localhost:9090:
```promql
# Request rate per second
rate(go_samples_http_requests_total[5m])

# 95th percentile latency
histogram_quantile(0.95, rate(go_samples_http_request_duration_seconds_bucket[5m]))

# Requests by status code
sum by (http_status_code) (go_samples_http_requests_total)
```

### Logs (Loki via Grafana)

Application logs are correlated with trace IDs. View in Grafana:
1. Go to Explore → Select "Loki" datasource
2. Query: `{service_name="go-otel-sample"}`
3. Click on a log line to see trace correlation

### Grafana Dashboard

A pre-configured dashboard is available at:
- Grafana → Dashboards → Go Samples Dashboard

Features:
- HTTP request rate gauge
- Request duration percentiles (p50, p95)
- Current task count

## Development

### Run Locally (without K8s)

```bash
# Start OTel Collector locally first (optional, or just run without telemetry)
docker run -p 4317:4317 otel/opentelemetry-collector-contrib:latest

# Run the application
make run
```

### Project Structure

```
go-otel-sample/
├── cmd/server/main.go           # Application entrypoint
├── internal/
│   ├── config/config.go         # Environment configuration
│   ├── handler/task.go          # HTTP handlers
│   ├── model/task.go            # Domain models
│   ├── repository/task.go       # Data access layer
│   └── telemetry/               # OpenTelemetry setup
│       ├── tracer.go            # Trace provider
│       ├── meter.go             # Metrics provider
│       └── logger.go            # Log provider (slog bridge)
├── k8s/
│   ├── base/                    # App Kubernetes manifests
│   └── observability/           # Observability stack manifests
├── Dockerfile                   # Multi-stage Docker build
├── Makefile                     # Development commands
└── README.md
```

### Useful Commands

```bash
make help           # Show all available commands
make build          # Build Go binary
make test           # Run tests
make docker-build   # Build Docker image
make k8s-deploy-all # Deploy everything
make k8s-delete-all # Clean up everything
make logs           # View application logs
make watch-pods     # Watch pod status
```

## Learning Resources

### Go
- [A Tour of Go](https://go.dev/tour/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go by Example](https://gobyexample.com/)
- [Chi Router Documentation](https://go-chi.io/)

### OpenTelemetry
- [OpenTelemetry Go Getting Started](https://opentelemetry.io/docs/languages/go/getting-started/)
- [OpenTelemetry Go SDK Documentation](https://pkg.go.dev/go.opentelemetry.io/otel)
- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)

### Kubernetes
- [Kubernetes Basics](https://kubernetes.io/docs/tutorials/kubernetes-basics/)
- [Kustomize Documentation](https://kustomize.io/)

## Cleanup

```bash
# Delete everything
make k8s-delete-all

# Or delete individually
make k8s-delete              # Delete app only
make k8s-observability-delete # Delete observability stack
```

## Troubleshooting

### Pods not starting

```bash
# Check pod status
kubectl get pods -n go-otel-sample

# Describe a failing pod
kubectl describe pod <pod-name> -n go-otel-sample

# Check logs
kubectl logs <pod-name> -n go-otel-sample
```

### No metrics/traces showing

1. Verify OTel Collector is running:
   ```bash
   kubectl logs -l app=otel-collector -n go-otel-sample
   ```

2. Check application logs for connection errors:
   ```bash
   kubectl logs -l app=go-otel-sample -n go-otel-sample
   ```

3. Ensure the OTLP endpoint is correct in the ConfigMap

### Image not found

Make sure to build the Docker image before deploying:
```bash
make docker-build
```

The deployment uses `imagePullPolicy: Never` to use local images with Docker Desktop.
