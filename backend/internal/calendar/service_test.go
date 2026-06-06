package calendar_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/calendar"
)

type fakeStore struct {
	created      *calendar.Draft
	replaceErr   error
	listed       []calendar.Event
	purgedUser   string
	purgeCalled  bool
	purgeReturns int
	purgeErr     error
}

func (f *fakeStore) Create(_ context.Context, userSubject string, d calendar.Draft) (calendar.Event, error) {
	f.created = &d
	return calendar.Event{ID: "e1", UserSubject: userSubject, Title: d.Title, AllDay: d.AllDay, Start: d.Start, End: d.End, Notes: d.Notes}, nil
}
func (f *fakeStore) ListInRange(context.Context, string, time.Time, time.Time) ([]calendar.Event, error) {
	return f.listed, nil
}
func (f *fakeStore) Replace(_ context.Context, _, id string, d calendar.Draft) (calendar.Event, error) {
	if f.replaceErr != nil {
		return calendar.Event{}, f.replaceErr
	}
	return calendar.Event{ID: id, Title: d.Title, Start: d.Start, End: d.End}, nil
}
func (f *fakeStore) Delete(context.Context, string, string) error { return nil }
func (f *fakeStore) PurgeUser(_ context.Context, userSubject string) (int, error) {
	f.purgeCalled = true
	f.purgedUser = userSubject
	return f.purgeReturns, f.purgeErr
}

func day(y int, m time.Month, d, h int) time.Time {
	return time.Date(y, m, d, h, 0, 0, 0, time.UTC)
}

func TestCreateValidation(t *testing.T) {
	tests := []struct {
		name    string
		draft   calendar.Draft
		wantErr error
	}{
		{
			name:  "timed event",
			draft: calendar.Draft{Title: "Standup", Start: day(2026, 6, 1, 9), End: day(2026, 6, 1, 10)},
		},
		{
			name:  "multi-day event (end on a later day)",
			draft: calendar.Draft{Title: "Conference", Start: day(2026, 6, 1, 9), End: day(2026, 6, 3, 17)},
		},
		{
			name:  "all-day event",
			draft: calendar.Draft{Title: "Holiday", AllDay: true, Start: day(2026, 6, 1, 0), End: day(2026, 6, 1, 0)},
		},
		{
			name:    "missing title",
			draft:   calendar.Draft{Start: day(2026, 6, 1, 9), End: day(2026, 6, 1, 10)},
			wantErr: calendar.ErrTitleRequired,
		},
		{
			name:    "end before start",
			draft:   calendar.Draft{Title: "Backwards", Start: day(2026, 6, 1, 10), End: day(2026, 6, 1, 9)},
			wantErr: calendar.ErrEndBeforeStart,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}
			_, err := calendar.NewService(store).Create(context.Background(), "user-1", tt.draft)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				if store.created != nil {
					t.Fatal("invalid draft must not reach the store")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if store.created == nil || store.created.Title != tt.draft.Title {
				t.Fatalf("store did not receive the draft: %+v", store.created)
			}
		})
	}
}

func TestListRejectsInvertedRange(t *testing.T) {
	_, err := calendar.NewService(&fakeStore{}).List(context.Background(), "user-1", day(2026, 6, 2, 0), day(2026, 6, 1, 0))
	if !errors.Is(err, calendar.ErrEndBeforeStart) {
		t.Fatalf("err = %v, want ErrEndBeforeStart", err)
	}
}

func TestReplacePropagatesNotFound(t *testing.T) {
	store := &fakeStore{replaceErr: calendar.ErrNotFound}
	_, err := calendar.NewService(store).Replace(context.Background(), "user-1", "missing",
		calendar.Draft{Title: "x", Start: day(2026, 6, 1, 9), End: day(2026, 6, 1, 10)})
	if !errors.Is(err, calendar.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestPurgeUserRejectsEmptySubject(t *testing.T) {
	store := &fakeStore{}
	n, err := calendar.NewService(store).PurgeUser(context.Background(), "")
	if !errors.Is(err, calendar.ErrUserSubjectRequired) {
		t.Fatalf("err = %v, want ErrUserSubjectRequired", err)
	}
	if n != 0 {
		t.Fatalf("count = %d, want 0", n)
	}
	if store.purgeCalled {
		t.Fatal("an empty subject must not reach the store")
	}
}

func TestPurgeUserDelegatesAndReturnsCount(t *testing.T) {
	store := &fakeStore{purgeReturns: 3}
	n, err := calendar.NewService(store).PurgeUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if n != 3 {
		t.Fatalf("count = %d, want 3", n)
	}
	if store.purgedUser != "user-1" {
		t.Fatalf("store purged %q, want user-1", store.purgedUser)
	}
}

func TestPurgeUserWrapsStoreError(t *testing.T) {
	sentinel := errors.New("db down")
	store := &fakeStore{purgeErr: sentinel}
	if _, err := calendar.NewService(store).PurgeUser(context.Background(), "user-1"); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want it to wrap the store error", err)
	}
}
