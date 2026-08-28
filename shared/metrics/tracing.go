package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"google.golang.org/grpc"
)

// Tracing owns the process-wide tracer provider.
type Tracing struct {
	Provider *sdktrace.TracerProvider
}

func NewTracing(ctx context.Context, serviceName, environment, endpoint string) (*Tracing, error) {
	attrs := []attribute.KeyValue{semconv.ServiceName(serviceName)}
	if environment != "" {
		attrs = append(attrs, semconv.DeploymentEnvironmentName(environment))
	}

	options := []sdktrace.TracerProviderOption{sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, attrs...))}
	if endpoint != "" {
		exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(endpoint), otlptracegrpc.WithInsecure())
		if err != nil {
			return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
		}
		options = append(options, sdktrace.WithBatcher(exporter))
	}

	provider := sdktrace.NewTracerProvider(options...)
	otel.SetTracerProvider(provider)
	return &Tracing{Provider: provider}, nil
}

func (t *Tracing) Shutdown(ctx context.Context) error { return t.Provider.Shutdown(ctx) }

// UnaryServerInterceptor creates RPC spans without installing a second gRPC
// stats handler. grpcmetrics remains the sole gRPC metrics implementation.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, span := otel.Tracer("cartix/grpc").Start(ctx, info.FullMethod)
		defer span.End()
		return handler(ctx, req)
	}
}
