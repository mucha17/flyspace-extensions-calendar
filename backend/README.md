# Calendar backend

The backend for the FlySpace Calendar extension. A single Go binary that owns the calendar's own
Postgres database and serves the events API. It is reached only through the platform's mediated
proxy: the platform attaches a short-lived delegated token, which this backend verifies against the
platform's published JWKS before acting on the user's behalf. No client ever calls this service
directly, and it never sees a raw user token.

## API

All data routes require the delegated bearer token and act on the token subject's events only.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/healthz` | liveness (public) |
| `GET` | `/events?from=<rfc3339>&to=<rfc3339>` | events overlapping the window |
| `POST` | `/events` | create an event |
| `PUT` | `/events/{id}` | replace an event |
| `DELETE` | `/events/{id}` | delete an event |

An event is `{ id, title, allDay, start, end, notes? }` with `start`/`end` in RFC 3339. `start` and
`end` may fall on different days (a multi-day event); an all-day event spans whole days. Errors use
`application/problem+json` with a stable `code`.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `CALENDAR_ADDR` | `:9090` | listen address |
| `CALENDAR_DATABASE_URL` | — (required) | Postgres connection string |
| `CALENDAR_CORE_JWKS_URL` | — (required) | platform JWKS endpoint (`…/.well-known/jwks.json`) |
| `CALENDAR_CORE_ISSUER` | — (required) | expected token issuer (the platform's public URL) |
| `CALENDAR_EXTENSION_ID` | `net.flytegration.calendar` | expected token audience |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — (optional) | OTLP/HTTP endpoint for traces + metrics; unset makes telemetry a no-op (trace ids still propagate) |

Migrations under `migrations/` are embedded and applied on startup.

## Develop

```bash
go build ./...
go test ./... -race                 # unit tests
go test -tags integration ./... -race   # adds the store tests (needs Docker for Postgres)
```
