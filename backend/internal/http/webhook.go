package http

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
)

// SubjectUserDeleted is core's GDPR teardown event. Frozen contract with core: core POSTs it to
// {endpoints.backend}/flyspace/events when a user is removed.
const SubjectUserDeleted = "flyspace.core.user.deleted"

// EventPurger applies a user-deletion teardown. *calendar.Service satisfies it.
type EventPurger interface {
	PurgeUser(ctx context.Context, userSubject string) (int, error)
}

// coreEvent is the body core POSTs for a fanned-out core event: subject identifies the event, data
// carries its payload, id is the delivery identifier (logs/idempotency).
type coreEvent struct {
	Subject string          `json:"subject"`
	ID      string          `json:"id"`
	Data    json.RawMessage `json:"data"`
}

type userDeletedData struct {
	Subject string `json:"subject"`
}

type webhookHandler struct{ purger EventPurger }

// receive handles a core event delivery. The request is already authenticated by the same delegated
// JWT check as the data routes (core's signature, verified against core's JWKS); the acting user is
// taken from the event body, not the token. Delivery is at-least-once, so handling is idempotent
// (PurgeUser is). A 2xx acks the delivery; any other status makes core redeliver.
func (h *webhookHandler) receive(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var evt coreEvent
	if !decode(w, r, &evt) {
		return
	}
	switch evt.Subject {
	case SubjectUserDeleted:
		var d userDeletedData
		if err := json.Unmarshal(evt.Data, &d); err != nil || d.Subject == "" {
			problem(w, stdhttp.StatusBadRequest, "validation.failed", "user.deleted event has no subject")
			return
		}
		if _, err := h.purger.PurgeUser(r.Context(), d.Subject); err != nil {
			// Transient: a non-2xx makes core redeliver, which is safe because PurgeUser is idempotent.
			problem(w, stdhttp.StatusInternalServerError, "internal", "teardown failed")
			return
		}
		w.WriteHeader(stdhttp.StatusNoContent)
	default:
		// Ack subjects we don't act on so core stops redelivering; we handle only what the manifest
		// subscribes to.
		w.WriteHeader(stdhttp.StatusNoContent)
	}
}
