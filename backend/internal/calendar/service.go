package calendar

import (
	"context"
	"fmt"
	"time"
)

// Service applies the calendar rules and delegates persistence to a Store. It acts only on behalf
// of the user subject the caller passes (resolved from the delegated token), so one user can never
// read or change another's events.
type Service struct {
	store Store
}

// NewService wires the calendar service to a store.
func NewService(store Store) *Service { return &Service{store: store} }

// Create validates and stores a new event for the user.
func (s *Service) Create(ctx context.Context, userSubject string, d Draft) (Event, error) {
	if err := d.validate(); err != nil {
		return Event{}, err
	}
	e, err := s.store.Create(ctx, userSubject, d)
	if err != nil {
		return Event{}, fmt.Errorf("create event: %w", err)
	}
	return e, nil
}

// List returns the user's events overlapping the [from, to) window.
func (s *Service) List(ctx context.Context, userSubject string, from, to time.Time) ([]Event, error) {
	if to.Before(from) {
		return nil, ErrEndBeforeStart
	}
	es, err := s.store.ListInRange(ctx, userSubject, from, to)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return es, nil
}

// Replace validates and overwrites the mutable fields of the user's event. It returns ErrNotFound
// if the event does not exist or belongs to another user.
func (s *Service) Replace(ctx context.Context, userSubject, id string, d Draft) (Event, error) {
	if err := d.validate(); err != nil {
		return Event{}, err
	}
	e, err := s.store.Replace(ctx, userSubject, id, d)
	if err != nil {
		return Event{}, fmt.Errorf("replace event: %w", err)
	}
	return e, nil
}

// Delete removes the user's event, or returns ErrNotFound.
func (s *Service) Delete(ctx context.Context, userSubject, id string) error {
	if err := s.store.Delete(ctx, userSubject, id); err != nil {
		return fmt.Errorf("delete event: %w", err)
	}
	return nil
}
