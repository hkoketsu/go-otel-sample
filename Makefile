.PHONY: build run test clean docker-build docker-run k8s-deploy k8s-delete k8s-observability k8s-observability-delete port-forward tidy fmt lint

# Application settings
APP_NAME := go-otel-sample
DOCKER_IMAGE := $(APP_NAME):latest

# Go settings
GO := go
GOFLAGS := -v

# Build the Go binary
build:
	$(GO) build $(GOFLAGS) -o bin/server ./cmd/server

# Run locally (requires OTel Collector running)
run:
	OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
	OTEL_SERVICE_NAME=$(APP_NAME) \
	ENVIRONMENT=development \
	$(GO) run ./cmd/server

# Run locally without OTel (for quick testing)
run-local:
	OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
	OTEL_SERVICE_NAME=$(APP_NAME) \
	ENVIRONMENT=development \
	$(GO) run ./cmd/server

# Run tests
test:
	$(GO) test -v ./...

# Run tests with coverage
test-coverage:
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Tidy go modules
tidy:
	$(GO) mod tidy

# Format code
fmt:
	$(GO) fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Build Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE) .

# Run Docker container locally
docker-run:
	docker run --rm -p 8080:8080 \
		-e OTEL_EXPORTER_OTLP_ENDPOINT=host.docker.internal:4317 \
		-e OTEL_SERVICE_NAME=$(APP_NAME) \
		-e ENVIRONMENT=development \
		$(DOCKER_IMAGE)

# Deploy observability stack to Kubernetes
k8s-observability:
	kubectl apply -k k8s/observability/

# Delete observability stack from Kubernetes
k8s-observability-delete:
	kubectl delete -k k8s/observability/ --ignore-not-found

# Deploy application to Kubernetes
k8s-deploy: docker-build
	kubectl apply -k k8s/base/

# Delete application from Kubernetes
k8s-delete:
	kubectl delete -k k8s/base/ --ignore-not-found

# Deploy everything (observability + app)
k8s-deploy-all: docker-build
	kubectl apply -k k8s/

# Delete everything
k8s-delete-all:
	kubectl delete -k k8s/ --ignore-not-found

# Port forward all observability UIs
port-forward:
	@echo "Starting port forwards..."
	@echo "Grafana:    http://localhost:3000"
	@echo "Jaeger:     http://localhost:16686"
	@echo "Prometheus: http://localhost:9090"
	@echo "App:        http://localhost:8080"
	@echo ""
	@echo "Press Ctrl+C to stop all port forwards"
	@kubectl port-forward svc/grafana 3000:3000 -n go-otel-sample & \
	kubectl port-forward svc/jaeger 16686:16686 -n go-otel-sample & \
	kubectl port-forward svc/prometheus 9090:9090 -n go-otel-sample & \
	kubectl port-forward svc/go-otel-sample 8080:8080 -n go-otel-sample & \
	wait

# Port forward just the app
port-forward-app:
	kubectl port-forward svc/go-otel-sample 8080:8080 -n go-otel-sample

# Watch pods
watch-pods:
	kubectl get pods -n go-otel-sample -w

# View app logs
logs:
	kubectl logs -f -l app=go-otel-sample -n go-otel-sample

# Describe app deployment
describe:
	kubectl describe deployment go-otel-sample -n go-otel-sample

# Quick API test
api-test:
	@echo "=== Health Check ==="
	curl -s http://localhost:8080/health | jq .
	@echo "\n=== Create Task ==="
	curl -s -X POST http://localhost:8080/api/v1/tasks \
		-H "Content-Type: application/json" \
		-d '{"title":"Learn Go","description":"Complete the Go tutorial"}' | jq .
	@echo "\n=== List Tasks ==="
	curl -s http://localhost:8080/api/v1/tasks | jq .

# Help
help:
	@echo "Available targets:"
	@echo "  build              - Build the Go binary"
	@echo "  run                - Run locally (requires OTel Collector)"
	@echo "  test               - Run tests"
	@echo "  test-coverage      - Run tests with coverage report"
	@echo "  clean              - Clean build artifacts"
	@echo "  tidy               - Tidy go modules"
	@echo "  fmt                - Format code"
	@echo "  lint               - Lint code"
	@echo "  docker-build       - Build Docker image"
	@echo "  docker-run         - Run Docker container locally"
	@echo "  k8s-observability  - Deploy observability stack"
	@echo "  k8s-deploy         - Deploy application"
	@echo "  k8s-deploy-all     - Deploy everything"
	@echo "  k8s-delete         - Delete application"
	@echo "  k8s-delete-all     - Delete everything"
	@echo "  port-forward       - Port forward all services"
	@echo "  port-forward-app   - Port forward just the app"
	@echo "  watch-pods         - Watch pod status"
	@echo "  logs               - View app logs"
	@echo "  api-test           - Quick API test"
