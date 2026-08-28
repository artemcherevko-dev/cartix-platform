package metrics

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mahboubii/grpcmetrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	prometheusexporter "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"google.golang.org/grpc/stats"
)

// Metrics contains the dedicated registry and HTTP server for one service.
type Metrics struct {
	MeterProvider    *sdkmetric.MeterProvider
	Exporter         *prometheusexporter.Exporter
	Registry         *prometheus.Registry
	GRPCStatsHandler stats.Handler
	server           *http.Server
}

func NewMetrics(serviceName, environment, port string) (*Metrics, error) {
	registry := prometheus.NewRegistry()
	exporter, err := prometheusexporter.New(prometheusexporter.WithRegisterer(registry))
	if err != nil {
		return nil, fmt.Errorf("create prometheus exporter: %w", err)
	}

	resourceAttrs := []attribute.KeyValue{semconv.ServiceName(serviceName)}
	if environment != "" {
		resourceAttrs = append(resourceAttrs, semconv.DeploymentEnvironmentName(environment))
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource.NewWithAttributes(semconv.SchemaURL, resourceAttrs...)),
		sdkmetric.WithReader(exporter),
	)
	// Keep the global provider aligned with the provider supplied to grpcmetrics.
	otel.SetMeterProvider(meterProvider)

	grpcHandler, err := grpcmetrics.NewServerHandler(
		grpcmetrics.WithMeterProvider(meterProvider),
		grpcmetrics.WithInstrumentLatency(true),
		grpcmetrics.WithInstrumentSizes(false),
	)
	if err != nil {
		_ = meterProvider.Shutdown(context.Background())
		return nil, fmt.Errorf("create grpc metrics handler: %w", err)
	}

	if port == "" {
		port = "9090"
	}

	return &Metrics{
		MeterProvider:    meterProvider,
		Exporter:         exporter,
		Registry:         registry,
		GRPCStatsHandler: grpcHandler,
		server: &http.Server{
			Addr:              ":" + port,
			Handler:           promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
			ReadHeaderTimeout: 5 * 1e9,
		},
	}, nil
}

// Start exposes only GET /metrics on the service-internal endpoint.
func (m *Metrics) Start() error {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{}))
	m.server.Handler = mux
	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// The caller owns service logging; preserve server startup errors through
			// the normal process logs without crashing an already-running gRPC server.
			fmt.Printf("metrics server stopped: %v\n", err)
		}
	}()
	return nil
}

func (m *Metrics) Shutdown(ctx context.Context) error {
	if err := m.server.Shutdown(ctx); err != nil {
		return err
	}
	return m.MeterProvider.Shutdown(ctx)
}
