package http

import (
	"context"
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mucha17/flyspace-extensions-calendar/backend/internal/calendar"
)

// EventService is the calendar behavior the handlers consume. *calendar.Service satisfies it.
type EventService interface {
	Create(ctx context.Context, userSubject string, d calendar.Draft) (calendar.Event, error)
	List(ctx context.Context, userSubject string, from, to time.Time) ([]calendar.Event, error)
	Replace(ctx context.Context, userSubject, id string, d calendar.Draft) (calendar.Event, error)
	Delete(ctx context.Context, userSubject, id string) error
}

type handlers struct{ events EventService }

// --- wire DTOs ---

type eventResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	AllDay    bool   `json:"allDay"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Notes     string `json:"notes,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func toResponse(e calendar.Event) eventResponse {
	return eventResponse{
		ID:        e.ID,
		Title:     e.Title,
		AllDay:    e.AllDay,
		Start:     e.Start.UTC().Format(time.RFC3339),
		End:       e.End.UTC().Format(time.RFC3339),
		Notes:     e.Notes,
		CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: e.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type eventRequest struct {
	Title  string `json:"title"`
	AllDay bool   `json:"allDay"`
	Start  string `json:"start"`
	End    string `json:"end"`
	Notes  string `json:"notes"`
}

// draft parses the request into a domain draft, reporting field errors for unparseable times.
func (req eventRequest) draft(w stdhttp.ResponseWriter) (calendar.Draft, bool) {
	start, ok := parseTime(w, "start", req.Start)
	if !ok {
		return calendar.Draft{}, false
	}
	end, ok := parseTime(w, "end", req.End)
	if !ok {
		return calendar.Draft{}, false
	}
	return calendar.Draft{Title: req.Title, AllDay: req.AllDay, Start: start, End: end, Notes: req.Notes}, true
}

func (h *handlers) list(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	from, ok := parseTime(w, "from", r.URL.Query().Get("from"))
	if !ok {
		return
	}
	to, ok := parseTime(w, "to", r.URL.Query().Get("to"))
	if !ok {
		return
	}
	events, err := h.events.List(r.Context(), userFrom(r.Context()), from, to)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]eventResponse, 0, len(events))
	for _, e := range events {
		out = append(out, toResponse(e))
	}
	writeJSON(w, stdhttp.StatusOK, out)
}

func (h *handlers) create(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req eventRequest
	if !decode(w, r, &req) {
		return
	}
	d, ok := req.draft(w)
	if !ok {
		return
	}
	e, err := h.events.Create(r.Context(), userFrom(r.Context()), d)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, stdhttp.StatusCreated, toResponse(e))
}

func (h *handlers) replace(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req eventRequest
	if !decode(w, r, &req) {
		return
	}
	d, ok := req.draft(w)
	if !ok {
		return
	}
	e, err := h.events.Replace(r.Context(), userFrom(r.Context()), chi.URLParam(r, "id"), d)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, stdhttp.StatusOK, toResponse(e))
}

func (h *handlers) delete(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if err := h.events.Delete(r.Context(), userFrom(r.Context()), chi.URLParam(r, "id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(stdhttp.StatusNoContent)
}

func health(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

// --- helpers ---

// writeError maps domain errors to the problem+json contract.
func writeError(w stdhttp.ResponseWriter, err error) {
	switch {
	case errors.Is(err, calendar.ErrNotFound):
		problem(w, stdhttp.StatusNotFound, "event.not_found", "event not found")
	case errors.Is(err, calendar.ErrTitleRequired):
		validationFailed(w, FieldError{Field: "title", Code: "required"})
	case errors.Is(err, calendar.ErrEndBeforeStart):
		validationFailed(w, FieldError{Field: "end", Code: "before_start"})
	default:
		problem(w, stdhttp.StatusInternalServerError, "internal", "internal server error")
	}
}

func parseTime(w stdhttp.ResponseWriter, field, value string) (time.Time, bool) {
	if value == "" {
		validationFailed(w, FieldError{Field: field, Code: "required"})
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		validationFailed(w, FieldError{Field: field, Code: "invalid"})
		return time.Time{}, false
	}
	return t, true
}

func decode(w stdhttp.ResponseWriter, r *stdhttp.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		problem(w, stdhttp.StatusBadRequest, "validation.failed", "invalid JSON body")
		return false
	}
	return true
}

func writeJSON(w stdhttp.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
