package config

import (
	"os"
)

// Config holds the application configuration.
type Config struct {
	// Server settings
	ServerPort string

	// OpenTelemetry settings
	OTLPEndpoint string
	ServiceName  string
	Environment  string
}

// Load returns configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		ServerPort:   getEnv("SERVER_PORT", "8080"),
		OTLPEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		ServiceName:  getEnv("OTEL_SERVICE_NAME", "go-samples"),
		Environment:  getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
