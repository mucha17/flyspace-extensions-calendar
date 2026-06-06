package http_test

import (
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
)

const userDeletedBody = `{"subject":"flyspace.core.user.deleted","id":"d1","data":{"subject":"user-9"}}`

func TestWebhookUserDeletedPurges(t *testing.T) {
	events := &fakeEvents{purgeCount: 3}
	rec := httptest.NewRecorder()
	server(events).ServeHTTP(rec, req("POST", "/flyspace/events", userDeletedBody, "good"))

	if rec.Code != stdhttp.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", rec.Code, rec.Body.String())
	}
	// The purged subject comes from the event body, not the token's subject.
	if events.purgedSubject != "user-9" {
		t.Fatalf("purged %q, want user-9", events.purgedSubject)
	}
}

func TestWebhookRequiresAuth(t *testing.T) {
	rec := httptest.NewRecorder()
	server(&fakeEvents{}).ServeHTTP(rec, req("POST", "/flyspace/events", userDeletedBody, ""))
	if rec.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestWebhookRejectsMissingUserSubject(t *testing.T) {
	events := &fakeEvents{}
	rec := httptest.NewRecorder()
	body := `{"subject":"flyspace.core.user.deleted","id":"d1","data":{}}`
	server(events).ServeHTTP(rec, req("POST", "/flyspace/events", body, "good"))

	if rec.Code != stdhttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	if events.purgedSubject != "" {
		t.Fatalf("a malformed payload must not purge, purged %q", events.purgedSubject)
	}
}

func TestWebhookAcksUnknownSubject(t *testing.T) {
	events := &fakeEvents{}
	rec := httptest.NewRecorder()
	body := `{"subject":"flyspace.core.thread.created","id":"d1","data":{}}`
	server(events).ServeHTTP(rec, req("POST", "/flyspace/events", body, "good"))

	if rec.Code != stdhttp.StatusNoContent {
		t.Fatalf("status = %d, want 204 (ack and ignore)", rec.Code)
	}
	if events.purgedSubject != "" {
		t.Fatalf("an unhandled subject must not purge, purged %q", events.purgedSubject)
	}
}

func TestWebhookPurgeErrorIsRetryable(t *testing.T) {
	events := &fakeEvents{purgeErr: errors.New("db down")}
	rec := httptest.NewRecorder()
	server(events).ServeHTTP(rec, req("POST", "/flyspace/events", userDeletedBody, "good"))

	// A non-2xx tells core to redeliver; the idempotent purge makes that safe.
	if rec.Code != stdhttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
