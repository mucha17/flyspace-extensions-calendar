package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"
)

// UserPurger erases everything this backend stored for a user and reports how many rows went. The
// calendar service satisfies it; the consumer declares it here, where it is used.
type UserPurger interface {
	PurgeUser(ctx context.Context, userSubject string) (int, error)
}

// UserDeletedEvent is the body of flyspace.core.user.deleted, published by core when a user is
// removed. Its shape is the platform contract.
type UserDeletedEvent struct {
	Subject string `json:"subject"`
}

// TeardownConsumer drives GDPR teardown: on core's user-deleted event it purges that user's events.
// PurgeUser is idempotent, so JetStream's at-least-once redelivery is safe.
type TeardownConsumer struct {
	conn   *Conn
	purger UserPurger
	log    *slog.Logger
}

// NewTeardownConsumer wires the consumer to the purger (the calendar service).
func NewTeardownConsumer(conn *Conn, purger UserPurger, log *slog.Logger) *TeardownConsumer {
	return &TeardownConsumer{conn: conn, purger: purger, log: log}
}

// Start creates the durable consumer on the core-owned CORE_EVENTS stream and begins consuming
// user-deleted events. The returned context is stopped by the caller on shutdown.
func (c *TeardownConsumer) Start(ctx context.Context) (jetstream.ConsumeContext, error) {
	cons, err := c.conn.js.CreateOrUpdateConsumer(ctx, StreamCoreEvents, jetstream.ConsumerConfig{
		Durable:       "calendar-teardown",
		FilterSubject: SubjectUserDeleted,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
	})
	if err != nil {
		return nil, fmt.Errorf("nats: teardown consumer: %w", err)
	}
	return cons.Consume(func(msg jetstream.Msg) { c.handle(ctx, msg) })
}

// ackAction is the teardown decision for a delivered message, separated from the JetStream message
// so the ack/nak/term policy is testable without a live broker.
type ackAction int

const (
	actionAck  ackAction = iota // processed; do not redeliver
	actionNak                   // transient failure; redeliver
	actionTerm                  // poison; never redeliver
)

func (c *TeardownConsumer) handle(ctx context.Context, msg jetstream.Msg) {
	switch c.apply(ctx, msg.Data()) {
	case actionTerm:
		_ = msg.Term()
	case actionNak:
		_ = msg.Nak()
	default:
		_ = msg.Ack()
	}
}

// apply decodes a user-deleted payload and purges the user. A malformed payload is poison (Term); a
// purge error is transient (Nak); success acks.
func (c *TeardownConsumer) apply(ctx context.Context, data []byte) ackAction {
	var evt UserDeletedEvent
	if err := json.Unmarshal(data, &evt); err != nil || evt.Subject == "" {
		c.log.Error("teardown: bad user.deleted payload", "err", err)
		return actionTerm
	}
	n, err := c.purger.PurgeUser(ctx, evt.Subject)
	if err != nil {
		c.log.Error("teardown: purge failed", "subject", evt.Subject, "err", err)
		return actionNak
	}
	c.log.Info("teardown: purged user events", "subject", evt.Subject, "deleted", n)
	return actionAck
}
