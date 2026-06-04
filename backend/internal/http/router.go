// Package http serves the calendar backend's API. Every data route is behind the delegated-token
// check; the public health probe is not.
package http

import (
	stdhttp "net/http"

	"github.com/go-chi/chi/v5"
)

// Deps wires the router. Events serves the data routes; Verifier authenticates them.
type Deps struct {
	Events   EventService
	Verifier Verifier
}

// NewRouter builds the calendar backend HTTP handler.
func NewRouter(d Deps) stdhttp.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", health)

	r.Group(func(api chi.Router) {
		api.Use(authenticate(d.Verifier))
		h := &handlers{events: d.Events}
		api.Get("/events", h.list)
		api.Post("/events", h.create)
		api.Put("/events/{id}", h.replace)
		api.Delete("/events/{id}", h.delete)
	})
	return r
}
