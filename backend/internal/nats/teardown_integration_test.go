//go:build integration

// Exercises the teardown consumer against a real JetStream broker: core's user-deleted event,
// published to CORE_EVENTS, must drive PurgeUser, and the ack/nak/term policy must hold against the
// real broker (poison terminated, transient failure redelivered).
package nats_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/testcontainers/testcontainers-go"
	tcnats "github.com/testcontainers/testcontainers-go/modules/nats"

	calnats "github.com/mucha17/flyspace-extensions-calendar/backend/internal/nats"
)

var natsURL string

func TestMain(m *testing.M) {
	ctx := context.Background()
	ctr, err := tcnats.Run(ctx, "nats:2.10") // the module enables JetStream by default
	if err != nil {
		panic(err)
	}
	natsURL, err = ctr.ConnectionString(ctx)
	if err != nil {
		panic(err)
	}

	code := m.Run()

	_ = testcontainers.TerminateContainer(ctr)
	os.Exit(code)
}

func discardLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// recordingPurger records every PurgeUser call and fails the first failFirst of them with a
// transient error, so redelivery can be exercised.
type recordingPurger struct {
	mu        sync.Mutex
	calls     []string
	failFirst int
	signal    chan struct{}
}

func (p *recordingPurger) PurgeUser(_ context.Context, userSubject string) (int, error) {
	p.mu.Lock()
	p.calls = append(p.calls, userSubject)
	n := len(p.calls)
	p.mu.Unlock()
	select {
	case p.signal <- struct{}{}:
	default:
	}
	if n <= p.failFirst {
		return 0, errors.New("transient")
	}
	return 1, nil
}

func (p *recordingPurger) subjects() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.calls...)
}

// freshStream resets CORE_EVENTS so each test starts clean — durable consumer state and undelivered
// messages must not leak between tests sharing the container. The test plays core's role here:
// in production core owns this stream.
func freshStream(t *testing.T, conn *calnats.Conn) {
	t.Helper()
	ctx := context.Background()
	js := conn.JetStream()
	_ = js.DeleteStream(ctx, calnats.StreamCoreEvents)
	if _, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     calnats.StreamCoreEvents,
		Subjects: []string{"flyspace.core.>", "flyspace.ext.>"},
	}); err != nil {
		t.Fatalf("create stream: %v", err)
	}
}

func startConsumer(t *testing.T, conn *calnats.Conn, p *recordingPurger) {
	t.Helper()
	cc, err := calnats.NewTeardownConsumer(conn, p, discardLog()).Start(context.Background())
	if err != nil {
		t.Fatalf("start consumer: %v", err)
	}
	t.Cleanup(cc.Stop)
}

func publish(t *testing.T, conn *calnats.Conn, body string) {
	t.Helper()
	if _, err := conn.JetStream().Publish(context.Background(), calnats.SubjectUserDeleted, []byte(body)); err != nil {
		t.Fatalf("publish: %v", err)
	}
}

func eventually(t *testing.T, within time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("condition not met within deadline")
}

func TestTeardownPurgesOnUserDeleted(t *testing.T) {
	conn, err := calnats.Connect(natsURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close()
	freshStream(t, conn)

	p := &recordingPurger{signal: make(chan struct{}, 8)}
	startConsumer(t, conn, p)

	publish(t, conn, `{"subject":"user-1"}`)

	select {
	case <-p.signal:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the purge")
	}
	if got := p.subjects(); len(got) != 1 || got[0] != "user-1" {
		t.Fatalf("purge calls = %v, want [user-1]", got)
	}
}

func TestTeardownTerminatesPoisonPayload(t *testing.T) {
	conn, err := calnats.Connect(natsURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close()
	freshStream(t, conn)

	p := &recordingPurger{signal: make(chan struct{}, 8)}
	startConsumer(t, conn, p)

	publish(t, conn, `not json`)

	// A poison payload is terminated, never redelivered, and never reaches the purger.
	select {
	case <-p.signal:
		t.Fatal("the purger ran on a poison payload")
	case <-time.After(2 * time.Second):
	}
	if got := p.subjects(); len(got) != 0 {
		t.Fatalf("purge calls = %v, want none", got)
	}
}

func TestTeardownRedeliversOnTransientError(t *testing.T) {
	conn, err := calnats.Connect(natsURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close()
	freshStream(t, conn)

	// Fail the first delivery; the Nak must redeliver and the retry must succeed.
	p := &recordingPurger{failFirst: 1, signal: make(chan struct{}, 8)}
	startConsumer(t, conn, p)

	publish(t, conn, `{"subject":"user-2"}`)

	eventually(t, 10*time.Second, func() bool { return len(p.subjects()) >= 2 })
	for _, s := range p.subjects() {
		if s != "user-2" {
			t.Fatalf("unexpected subject %q in %v", s, p.subjects())
		}
	}
}
