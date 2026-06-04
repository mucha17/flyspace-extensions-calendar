CREATE SCHEMA IF NOT EXISTS calendar;

CREATE TABLE IF NOT EXISTS calendar.event (
    id           text        PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_subject text        NOT NULL,
    title        text        NOT NULL,
    all_day      boolean     NOT NULL DEFAULT false,
    starts_at    timestamptz NOT NULL,
    ends_at      timestamptz NOT NULL,
    notes        text        NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- Range queries are always scoped to one user and ordered by start.
CREATE INDEX IF NOT EXISTS idx_event_user_range ON calendar.event (user_subject, starts_at, ends_at);
