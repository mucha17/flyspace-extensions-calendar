// Package calendar is the calendar domain: events and the rules for creating them. It declares the
// Store interface it needs and is free of HTTP and database concerns.
package calendar

import (
	"context"
	"errors"
	"time"
)

// Errors callers branch on.
var (
	ErrNotFound       = errors.New("calendar: event not found")
	ErrTitleRequired  = errors.New("calendar: title is required")
	ErrEndBeforeStart = errors.New("calendar: end is before start")
)

// Event is a calendar entry owned by one user. An all-day event spans whole days; a timed event
// runs from Start to End. Start and End may fall on different days — a multi-day event.
type Event struct {
	ID          string
	UserSubject string
	Title       string
	AllDay      bool
	Start       time.Time
	End         time.Time
	Notes       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Draft is the mutable shape a caller supplies to create or replace an event.
type Draft struct {
	Title  string
	AllDay bool
	Start  time.Time
	End    time.Time
	Notes  string
}

func (d Draft) validate() error {
	if d.Title == "" {
		return ErrTitleRequired
	}
	if d.End.Before(d.Start) {
		return ErrEndBeforeStart
	}
	return nil
}

// Store persists events for a user. The store package implements it against Postgres.
type Store interface {
	Create(ctx context.Context, userSubject string, d Draft) (Event, error)
	ListInRange(ctx context.Context, userSubject string, from, to time.Time) ([]Event, error)
	Replace(ctx context.Context, userSubject, id string, d Draft) (Event, error)
	Delete(ctx context.Context, userSubject, id string) error
}
