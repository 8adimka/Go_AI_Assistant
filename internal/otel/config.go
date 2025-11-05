package otel

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitOpenTelemetry initializes OpenTelemetry with Prometheus exporter
func InitOpenTelemetry(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
			semconv.DeploymentEnvironment("development"),
		),
	)
	if err != nil {
		return nil, err
	}

	// Initialize metrics
	metricExporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metricExporter),
	)
	otel.SetMeterProvider(meterProvider)

	// Initialize tracing
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(traceProvider)

	// Return shutdown function
	shutdown := func(ctx context.Context) error {
		var errs []error

		if err := meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}

		if err := traceProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}

		if len(errs) > 0 {
			slog.ErrorContext(ctx, "Failed to shutdown OpenTelemetry providers", "errors", errs)
			return errs[0]
		}

		slog.InfoContext(ctx, "OpenTelemetry providers shutdown successfully")
		return nil
	}

	slog.InfoContext(ctx, "OpenTelemetry initialized successfully", "service", serviceName)
	return shutdown, nil
}

// GetTracer returns a tracer for the service
func GetTracer() interface{} {
	return otel.Tracer("github.com/8adimka/Go_AI_Assistant")
}

// GetMeter returns a meter for the service
func GetMeter() interface{} {
	return otel.Meter("github.com/8adimka/Go_AI_Assistant")
}
