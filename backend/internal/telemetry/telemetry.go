// Package telemetry sets up OpenTelemetry for the calendar backend. It always installs the W3C
// trace-context propagator, so a user action's trace (browser → core → this backend) stays one
// connected trace even with no local exporter. The OTLP/HTTP tracer and meter providers are
// installed only when an endpoint is configured; otherwise telemetry is a no-op.
package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Shutdown flushes and stops the providers. It is a no-op when no exporter was configured.
type Shutdown func(context.Context) error

// Init installs the propagator and, when otlpEndpoint is non-empty, OTLP/HTTP tracer and meter
// providers tagged with serviceName. The returned Shutdown should be deferred at the composition
// root.
func Init(ctx context.Context, serviceName, otlpEndpoint string) (Shutdown, error) {
	// The propagator is always set: trace ids must flow even with no local exporter.
	otel.SetTextMapPropagator(propagation.TraceContext{})

	if otlpEndpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(serviceName)))
	if err != nil {
		return nil, fmt.Errorf("telemetry: resource: %w", err)
	}

	traceExporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(otlpEndpoint))
	if err != nil {
		return nil, fmt.Errorf("telemetry: trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	metricExporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpointURL(otlpEndpoint))
	if err != nil {
		return nil, fmt.Errorf("telemetry: metric exporter: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	return func(ctx context.Context) error {
		if err := tp.Shutdown(ctx); err != nil {
			return err
		}
		return mp.Shutdown(ctx)
	}, nil
}
