package telemetry_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/telemetry"
)

func TestInitWithoutEndpointIsNoop(t *testing.T) {
	shutdown, err := telemetry.Init(context.Background(), "test", "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown must be non-nil even with no exporter")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("no-op shutdown returned err: %v", err)
	}
}

func TestInitAlwaysInstallsTraceContextPropagator(t *testing.T) {
	// Even with no exporter, trace ids must propagate so the backend joins core's trace.
	if _, err := telemetry.Init(context.Background(), "test", ""); err != nil {
		t.Fatalf("init: %v", err)
	}
	carrier := propagation.MapCarrier{
		"traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
	}
	ctx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		t.Fatal("propagator did not extract a valid span context; W3C trace-context not installed")
	}
}
