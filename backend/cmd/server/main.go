// Command server is the calendar extension's backend: a single Go binary that owns the calendar's
// Postgres database and serves the events API. It is reached only through core's mediated proxy,
// which attaches a delegated token this backend verifies against core's JWKS.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/auth"
	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/calendar"
	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/config"
	calhttp "github.com/mucha17/flyspace-extensions-calendar/backend/internal/http"
	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/store"
	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/telemetry"
)

const serviceName = "flyspace-ext-calendar"

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("calendar backend exited", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Always installs the W3C propagator so the trace joins core's; the OTLP exporter is set up only
	// when OTEL_EXPORTER_OTLP_ENDPOINT is configured, otherwise this is a no-op.
	shutdownTelemetry, err := telemetry.Init(ctx, serviceName, cfg.OTLPEndpoint)
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownTelemetry(shutdownCtx)
	}()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()
	if err := store.Migrate(ctx, pool); err != nil {
		return err
	}

	verifier, err := auth.NewVerifier(ctx, cfg.CoreJWKSURL, cfg.CoreIssuer, cfg.ExtensionID)
	if err != nil {
		return err
	}

	events := calendar.NewService(store.NewPgEventStore(pool))
	// GDPR teardown is delivered by core over HTTP: core POSTs flyspace.core.user.deleted to the
	// /flyspace/events webhook, which drives events.PurgeUser. (The NATS consumer in internal/nats
	// is retained for the deferred broker fast path but is not wired.)
	// otelhttp creates a server span per request and extracts incoming W3C trace context, so this
	// backend's spans become children of core's.
	handler := otelhttp.NewHandler(
		calhttp.NewRouter(calhttp.Deps{Events: events, CoreEvents: events, Verifier: verifier}),
		serviceName,
	)

	srv := &stdhttp.Server{Addr: cfg.Addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Info("calendar backend listening", "addr", cfg.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}
