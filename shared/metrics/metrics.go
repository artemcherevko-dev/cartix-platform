// Package metrics provides the shared observability primitives used by Cartix
// services. gRPC metrics are intentionally produced only by grpcmetrics.
package metrics

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc/stats"
)

// Config is the common, environment-driven telemetry configuration.
type Config struct {
	ServiceName  string
	Environment  string
	MetricsPort  string
	OTLPEndpoint string
}

// Telemetry owns all telemetry resources for one service.
type Telemetry struct {
	Metrics *Metrics
	Tracing *Tracing
}

// ConfigFromEnv follows the project's existing environment based configuration.
// METRICS_PORT defaults to 9090 when it is not set.
func ConfigFromEnv(serviceName string) Config {
	return Config{
		ServiceName:  serviceName,
		Environment:  os.Getenv("ENVIRONMENT"),
		MetricsPort:  os.Getenv("METRICS_PORT"),
		OTLPEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}
}

// New initializes metrics and tracing, but does not expose the metrics HTTP
// endpoint until Start is called.
func New(ctx context.Context, cfg Config) (*Telemetry, error) {
	metrics, err := NewMetrics(cfg.ServiceName, cfg.Environment, cfg.MetricsPort)
	if err != nil {
		return nil, fmt.Errorf("initialize metrics: %w", err)
	}

	tracing, err := NewTracing(ctx, cfg.ServiceName, cfg.Environment, cfg.OTLPEndpoint)
	if err != nil {
		_ = metrics.Shutdown(context.Background())
		return nil, fmt.Errorf("initialize tracing: %w", err)
	}

	return &Telemetry{Metrics: metrics, Tracing: tracing}, nil
}

// GRPCStatsHandler is the single source of gRPC server metrics.
func (t *Telemetry) GRPCStatsHandler() stats.Handler {
	return t.Metrics.GRPCStatsHandler
}

func (t *Telemetry) Start() error { return t.Metrics.Start() }

// Shutdown releases resources in reverse dependency order.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	if err := t.Metrics.Shutdown(ctx); err != nil {
		return err
	}
	return t.Tracing.Shutdown(ctx)
}
