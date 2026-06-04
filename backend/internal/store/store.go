// Package store persists calendar events in Postgres with pgx. It maps database rows to domain
// types so the calendar package never imports a database package.
package store

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/calendar"
	"github.com/mucha17/flyspace-extensions-calendar/backend/migrations"
)

// PgEventStore is the Postgres-backed calendar.Store.
type PgEventStore struct{ pool *pgxpool.Pool }

// NewPgEventStore builds the store over a pgx pool.
func NewPgEventStore(pool *pgxpool.Pool) *PgEventStore { return &PgEventStore{pool: pool} }

const eventColumns = `id, user_subject, title, all_day, starts_at, ends_at, notes, created_at, updated_at`

func (s *PgEventStore) Create(ctx context.Context, userSubject string, d calendar.Draft) (calendar.Event, error) {
	row := s.pool.QueryRow(ctx,
		`INSERT INTO calendar.event (user_subject, title, all_day, starts_at, ends_at, notes)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+eventColumns,
		userSubject, d.Title, d.AllDay, d.Start, d.End, d.Notes)
	e, err := scanEvent(row)
	if err != nil {
		return calendar.Event{}, fmt.Errorf("insert event: %w", err)
	}
	return e, nil
}

func (s *PgEventStore) ListInRange(ctx context.Context, userSubject string, from, to time.Time) ([]calendar.Event, error) {
	// An event overlaps [from, to) when it starts before `to` and ends at or after `from`.
	rows, err := s.pool.Query(ctx,
		`SELECT `+eventColumns+`
		 FROM calendar.event
		 WHERE user_subject = $1 AND starts_at < $3 AND ends_at >= $2
		 ORDER BY starts_at`,
		userSubject, from, to)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	out := make([]calendar.Event, 0)
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}
	return out, nil
}

func (s *PgEventStore) Replace(ctx context.Context, userSubject, id string, d calendar.Draft) (calendar.Event, error) {
	row := s.pool.QueryRow(ctx,
		`UPDATE calendar.event
		 SET title = $3, all_day = $4, starts_at = $5, ends_at = $6, notes = $7, updated_at = now()
		 WHERE id = $1 AND user_subject = $2
		 RETURNING `+eventColumns,
		id, userSubject, d.Title, d.AllDay, d.Start, d.End, d.Notes)
	e, err := scanEvent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return calendar.Event{}, calendar.ErrNotFound
	}
	if err != nil {
		return calendar.Event{}, fmt.Errorf("update event: %w", err)
	}
	return e, nil
}

func (s *PgEventStore) Delete(ctx context.Context, userSubject, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM calendar.event WHERE id = $1 AND user_subject = $2`, id, userSubject)
	if err != nil {
		return fmt.Errorf("delete event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return calendar.ErrNotFound
	}
	return nil
}

// rowScanner is satisfied by both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanEvent(r rowScanner) (calendar.Event, error) {
	var e calendar.Event
	if err := r.Scan(&e.ID, &e.UserSubject, &e.Title, &e.AllDay, &e.Start, &e.End, &e.Notes, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return calendar.Event{}, err
	}
	return e, nil
}

// Migrate applies every embedded "*.up.sql" file not yet recorded in schema_migrations, each in its
// own transaction. It is forward-only and safe to run on every startup.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("migrate: tracking table: %w", err)
	}
	names, err := fs.Glob(migrations.FS, "*.up.sql")
	if err != nil {
		return fmt.Errorf("migrate: list: %w", err)
	}
	sort.Strings(names)
	for _, name := range names {
		version := strings.TrimSuffix(name, ".up.sql")
		var applied bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&applied); err != nil {
			return fmt.Errorf("migrate: check %s: %w", version, err)
		}
		if applied {
			continue
		}
		sqlBytes, err := migrations.FS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", name, err)
		}
		if err := applyMigration(ctx, pool, version, string(sqlBytes)); err != nil {
			return err
		}
	}
	return nil
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, version, body string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("migrate %s: begin: %w", version, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, body); err != nil {
		return fmt.Errorf("migrate %s: apply: %w", version, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
		return fmt.Errorf("migrate %s: record: %w", version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("migrate %s: commit: %w", version, err)
	}
	return nil
}
