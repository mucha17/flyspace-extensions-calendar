package nats

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
)

type fakePurger struct {
	called  bool
	subject string
	ret     int
	err     error
}

func (f *fakePurger) PurgeUser(_ context.Context, userSubject string) (int, error) {
	f.called = true
	f.subject = userSubject
	return f.ret, f.err
}

func discardLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestApplyAcksAndPurgesOnValidEvent(t *testing.T) {
	p := &fakePurger{ret: 2}
	c := NewTeardownConsumer(nil, p, discardLog())

	if got := c.apply(context.Background(), []byte(`{"subject":"user-1"}`)); got != actionAck {
		t.Fatalf("action = %v, want actionAck", got)
	}
	if !p.called || p.subject != "user-1" {
		t.Fatalf("purger called=%v subject=%q, want it called with user-1", p.called, p.subject)
	}
}

func TestApplyTermsPoisonPayload(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"malformed json", `not json`},
		{"empty subject", `{"subject":""}`},
		{"missing subject", `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &fakePurger{}
			c := NewTeardownConsumer(nil, p, discardLog())

			if got := c.apply(context.Background(), []byte(tt.data)); got != actionTerm {
				t.Fatalf("action = %v, want actionTerm", got)
			}
			if p.called {
				t.Fatal("a poison payload must not reach the purger")
			}
		})
	}
}

func TestApplyNaksOnTransientPurgeError(t *testing.T) {
	p := &fakePurger{err: errors.New("db down")}
	c := NewTeardownConsumer(nil, p, discardLog())

	if got := c.apply(context.Background(), []byte(`{"subject":"user-1"}`)); got != actionNak {
		t.Fatalf("action = %v, want actionNak", got)
	}
}
