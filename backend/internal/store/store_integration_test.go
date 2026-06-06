//go:build integration

package store_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/calendar"
	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/store"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()
	ctr, err := postgres.Run(ctx, "postgres:16",
		postgres.WithDatabase("calendar"),
		postgres.WithUsername("calendar"),
		postgres.WithPassword("dev"),
		testcontainers.WithWaitStrategy(wait.ForAll(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second),
			// Postgres opens the port mid-init and restarts the server, so a port check alone races
			// the real readiness; wait for the readiness log to appear the second time.
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second),
		)),
	)
	if err != nil {
		panic(err)
	}
	url, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}
	if pool, err = pgxpool.New(ctx, url); err != nil {
		panic(err)
	}
	if err := store.Migrate(ctx, pool); err != nil {
		panic(err)
	}

	code := m.Run()

	pool.Close()
	_ = testcontainers.TerminateContainer(ctr)
	os.Exit(code)
}

func day(d, h int) time.Time { return time.Date(2026, 6, d, h, 0, 0, 0, time.UTC) }

func TestEventLifecycle(t *testing.T) {
	ctx := context.Background()
	s := store.NewPgEventStore(pool)
	const user = "user-lifecycle"

	created, err := s.Create(ctx, user, calendar.Draft{
		Title: "Standup", Start: day(1, 9), End: day(1, 10), Notes: "daily",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" || created.CreatedAt.IsZero() {
		t.Fatalf("DB did not assign id/timestamps: %+v", created)
	}

	// In range.
	got, err := s.ListInRange(ctx, user, day(1, 0), day(2, 0))
	if err != nil || len(got) != 1 || got[0].ID != created.ID {
		t.Fatalf("ListInRange = (%d events, %v), want the created event", len(got), err)
	}
	// Out of range.
	if got, _ := s.ListInRange(ctx, user, day(5, 0), day(6, 0)); len(got) != 0 {
		t.Fatalf("ListInRange out of window returned %d, want 0", len(got))
	}

	replaced, err := s.Replace(ctx, user, created.ID, calendar.Draft{Title: "Standup (moved)", Start: day(1, 11), End: day(1, 12)})
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}
	if replaced.Title != "Standup (moved)" || !replaced.UpdatedAt.After(created.UpdatedAt) {
		t.Fatalf("Replace did not update: %+v", replaced)
	}

	if err := s.Delete(ctx, user, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := s.Delete(ctx, user, created.ID); !errors.Is(err, calendar.ErrNotFound) {
		t.Fatalf("second Delete = %v, want ErrNotFound", err)
	}
}

func TestMultiDayOverlap(t *testing.T) {
	ctx := context.Background()
	s := store.NewPgEventStore(pool)
	const user = "user-multiday"

	// A conference spanning June 1–3.
	ev, err := s.Create(ctx, user, calendar.Draft{Title: "Conference", Start: day(1, 9), End: day(3, 17)})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// A window touching only June 2 must still see the multi-day event.
	got, err := s.ListInRange(ctx, user, day(2, 0), day(2, 23))
	if err != nil || len(got) != 1 || got[0].ID != ev.ID {
		t.Fatalf("multi-day event not returned for an inner-day window: (%d, %v)", len(got), err)
	}
}

func TestPurgeUserRemovesOnlyThatUsersEvents(t *testing.T) {
	ctx := context.Background()
	s := store.NewPgEventStore(pool)
	const victim, bystander = "purge-victim", "purge-bystander"

	for _, d := range []calendar.Draft{
		{Title: "A", Start: day(20, 9), End: day(20, 10)},
		{Title: "B", Start: day(21, 9), End: day(21, 10)},
	} {
		if _, err := s.Create(ctx, victim, d); err != nil {
			t.Fatalf("seed victim: %v", err)
		}
	}
	if _, err := s.Create(ctx, bystander, calendar.Draft{Title: "Keep", Start: day(20, 9), End: day(20, 10)}); err != nil {
		t.Fatalf("seed bystander: %v", err)
	}

	n, err := s.PurgeUser(ctx, victim)
	if err != nil {
		t.Fatalf("PurgeUser: %v", err)
	}
	if n != 2 {
		t.Fatalf("purged %d, want 2", n)
	}

	// The victim's events are gone; the bystander's remain.
	if got, _ := s.ListInRange(ctx, victim, day(1, 0), day(28, 0)); len(got) != 0 {
		t.Fatalf("victim still has %d events, want 0", len(got))
	}
	if got, _ := s.ListInRange(ctx, bystander, day(1, 0), day(28, 0)); len(got) != 1 {
		t.Fatalf("bystander has %d events, want 1", len(got))
	}

	// Idempotent: purging again removes nothing and is not an error.
	if n, err := s.PurgeUser(ctx, victim); err != nil || n != 0 {
		t.Fatalf("second PurgeUser = (%d, %v), want (0, nil)", n, err)
	}
}

func TestUserIsolation(t *testing.T) {
	ctx := context.Background()
	s := store.NewPgEventStore(pool)

	mine, err := s.Create(ctx, "owner", calendar.Draft{Title: "Private", Start: day(10, 9), End: day(10, 10)})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Another user cannot see it.
	if got, _ := s.ListInRange(ctx, "intruder", day(10, 0), day(11, 0)); len(got) != 0 {
		t.Fatalf("intruder saw %d events, want 0", len(got))
	}
	// Another user cannot replace or delete it.
	if _, err := s.Replace(ctx, "intruder", mine.ID, calendar.Draft{Title: "hijack", Start: day(10, 9), End: day(10, 10)}); !errors.Is(err, calendar.ErrNotFound) {
		t.Fatalf("intruder Replace = %v, want ErrNotFound", err)
	}
	if err := s.Delete(ctx, "intruder", mine.ID); !errors.Is(err, calendar.ErrNotFound) {
		t.Fatalf("intruder Delete = %v, want ErrNotFound", err)
	}
}
