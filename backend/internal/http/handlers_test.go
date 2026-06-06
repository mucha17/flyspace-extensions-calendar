package http_test

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/calendar"
	calhttp "github.com/mucha17/flyspace-extensions-calendar/backend/internal/http"
)

type fakeEvents struct {
	created       *calendar.Draft
	createdBy     string
	replaceErr    error
	deleteErr     error
	purgedSubject string
	purgeCount    int
	purgeErr      error
}

func (f *fakeEvents) Create(_ context.Context, userSubject string, d calendar.Draft) (calendar.Event, error) {
	f.created, f.createdBy = &d, userSubject
	return calendar.Event{ID: "e1", Title: d.Title, AllDay: d.AllDay, Start: d.Start, End: d.End}, nil
}
func (f *fakeEvents) List(context.Context, string, time.Time, time.Time) ([]calendar.Event, error) {
	return []calendar.Event{{ID: "e1", Title: "Standup", Start: time.Now(), End: time.Now()}}, nil
}
func (f *fakeEvents) Replace(_ context.Context, _, id string, d calendar.Draft) (calendar.Event, error) {
	if f.replaceErr != nil {
		return calendar.Event{}, f.replaceErr
	}
	return calendar.Event{ID: id, Title: d.Title, Start: d.Start, End: d.End}, nil
}
func (f *fakeEvents) Delete(context.Context, string, string) error { return f.deleteErr }
func (f *fakeEvents) PurgeUser(_ context.Context, userSubject string) (int, error) {
	f.purgedSubject = userSubject
	if f.purgeErr != nil {
		return 0, f.purgeErr
	}
	return f.purgeCount, nil
}

// fakeVerifier accepts the literal token "good" as user-1 and rejects everything else.
type fakeVerifier struct{}

func (fakeVerifier) Subject(token string) (string, error) {
	if token == "good" {
		return "user-1", nil
	}
	return "", stdhttp.ErrAbortHandler
}

func server(events *fakeEvents) stdhttp.Handler {
	return calhttp.NewRouter(calhttp.Deps{Events: events, CoreEvents: events, Verifier: fakeVerifier{}})
}

func req(method, path, body, token string) *stdhttp.Request {
	var r *stdhttp.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return r
}

func TestCreateEvent(t *testing.T) {
	events := &fakeEvents{}
	rec := httptest.NewRecorder()
	server(events).ServeHTTP(rec, req("POST", "/events",
		`{"title":"Standup","start":"2026-06-01T09:00:00Z","end":"2026-06-01T10:00:00Z"}`, "good"))

	if rec.Code != stdhttp.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	if events.created == nil || events.created.Title != "Standup" {
		t.Fatalf("draft not forwarded: %+v", events.created)
	}
	if events.createdBy != "user-1" {
		t.Fatalf("createdBy = %q, want the token's subject user-1", events.createdBy)
	}
}

func TestCreateInvalidTimeIsUnprocessable(t *testing.T) {
	rec := httptest.NewRecorder()
	server(&fakeEvents{}).ServeHTTP(rec, req("POST", "/events", `{"title":"x","start":"nope","end":"2026-06-01T10:00:00Z"}`, "good"))
	if rec.Code != stdhttp.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"start"`) {
		t.Fatalf("expected a field error for start: %s", rec.Body.String())
	}
}

func TestListRequiresRange(t *testing.T) {
	rec := httptest.NewRecorder()
	server(&fakeEvents{}).ServeHTTP(rec, req("GET", "/events", "", "good"))
	if rec.Code != stdhttp.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (missing from/to)", rec.Code)
	}
}

func TestListReturnsEvents(t *testing.T) {
	rec := httptest.NewRecorder()
	server(&fakeEvents{}).ServeHTTP(rec, req("GET", "/events?from=2026-06-01T00:00:00Z&to=2026-06-30T00:00:00Z", "", "good"))
	if rec.Code != stdhttp.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"Standup"`) {
		t.Fatalf("expected the event in the body: %s", rec.Body.String())
	}
}

func TestReplaceNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	server(&fakeEvents{replaceErr: calendar.ErrNotFound}).ServeHTTP(rec,
		req("PUT", "/events/missing", `{"title":"x","start":"2026-06-01T09:00:00Z","end":"2026-06-01T10:00:00Z"}`, "good"))
	if rec.Code != stdhttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"event.not_found"`) {
		t.Fatalf("expected event.not_found: %s", rec.Body.String())
	}
}

func TestDeleteEvent(t *testing.T) {
	rec := httptest.NewRecorder()
	server(&fakeEvents{}).ServeHTTP(rec, req("DELETE", "/events/e1", "", "good"))
	if rec.Code != stdhttp.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestUnauthenticated(t *testing.T) {
	// No token at all.
	rec := httptest.NewRecorder()
	server(&fakeEvents{}).ServeHTTP(rec, req("GET", "/events?from=2026-06-01T00:00:00Z&to=2026-06-02T00:00:00Z", "", ""))
	if rec.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"auth.required"`) {
		t.Fatalf("expected auth.required: %s", rec.Body.String())
	}

	// A token the verifier rejects.
	rec = httptest.NewRecorder()
	server(&fakeEvents{}).ServeHTTP(rec, req("GET", "/events?from=2026-06-01T00:00:00Z&to=2026-06-02T00:00:00Z", "", "bad"))
	if rec.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"auth.invalid"`) {
		t.Fatalf("expected auth.invalid: %s", rec.Body.String())
	}
}

func TestHealthIsPublic(t *testing.T) {
	rec := httptest.NewRecorder()
	server(&fakeEvents{}).ServeHTTP(rec, req("GET", "/healthz", "", ""))
	if rec.Code != stdhttp.StatusOK {
		t.Fatalf("status = %d, want 200 (health needs no auth)", rec.Code)
	}
}
