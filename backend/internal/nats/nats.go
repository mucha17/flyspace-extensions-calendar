// Package nats is the calendar backend's opt-in NATS fast path. It is dormant unless a broker URL is
// configured; today it carries a single consumer — the GDPR teardown trigger. Core publishes
// flyspace.core.user.deleted on the CORE_EVENTS stream when a user is removed, and this consumer
// reacts by purging that user's events. The browser never connects here — only this backend does,
// and only when the platform provisions broker access.
package nats

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Frozen surface shared with core: the stream and subject the teardown trigger lives on.
const (
	StreamCoreEvents   = "CORE_EVENTS"
	SubjectUserDeleted = "flyspace.core.user.deleted"
)

// Conn wraps the NATS connection and its JetStream context.
type Conn struct {
	nc *nats.Conn
	js jetstream.JetStream
}

// Connect dials NATS and builds the JetStream context. opts lets callers (and tests) add dial
// options such as credentials once the platform provisions them.
func Connect(url string, opts ...nats.Option) (*Conn, error) {
	opts = append([]nats.Option{
		nats.Name("flyspace-ext-calendar"),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2 * time.Second),
	}, opts...)
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats: connect: %w", err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats: jetstream: %w", err)
	}
	return &Conn{nc: nc, js: js}, nil
}

// JetStream exposes the JetStream context for consumers in this package.
func (c *Conn) JetStream() jetstream.JetStream { return c.js }

// Close closes the underlying connection.
func (c *Conn) Close() { c.nc.Close() }
