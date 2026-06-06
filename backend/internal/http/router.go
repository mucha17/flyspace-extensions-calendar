// Package http serves the calendar backend's API. Every data route is behind the delegated-token
// check; the public health probe is not.
package http

import (
	stdhttp "net/http"

	"github.com/go-chi/chi/v5"
)

// Deps wires the router. Events serves the data routes, CoreEvents applies core's webhook-delivered
// lifecycle events (GDPR teardown), and Verifier authenticates both.
type Deps struct {
	Events     EventService
	CoreEvents EventPurger
	Verifier   Verifier
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

		// Core fans its lifecycle events (e.g. GDPR teardown) here, authenticated by the same
		// delegated-token check as the data routes. Reserved path, separate from the proxied API.
		wh := &webhookHandler{purger: d.CoreEvents}
		api.Post("/flyspace/events", wh.receive)
	})
	return r
}
