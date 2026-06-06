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

	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/auth"
	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/calendar"
	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/config"
	calhttp "github.com/mucha17/flyspace-extensions-calendar/backend/internal/http"
	calnats "github.com/mucha17/flyspace-extensions-calendar/backend/internal/nats"
	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/store"
)

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

	// Opt-in NATS fast path: the GDPR teardown consumer. It is dormant unless a broker URL is
	// configured, and a NATS problem never blocks the HTTP backend (its primary function) — like the
	// JWKS verifier, it degrades rather than failing boot.
	if cfg.NatsURL != "" {
		if conn, err := calnats.Connect(cfg.NatsURL); err != nil {
			log.Error("nats connect failed; GDPR teardown consumer disabled", "err", err)
		} else {
			defer conn.Close()
			if cc, err := calnats.NewTeardownConsumer(conn, events, log).Start(ctx); err != nil {
				log.Error("teardown consumer start failed; GDPR teardown disabled", "err", err)
			} else {
				defer cc.Stop()
				log.Info("GDPR teardown consumer started", "subject", calnats.SubjectUserDeleted)
			}
		}
	} else {
		log.Info("nats not configured; GDPR teardown consumer disabled")
	}

	handler := calhttp.NewRouter(calhttp.Deps{Events: events, Verifier: verifier})

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
