# Copilot Instructions

## Project Overview

**Timeslot** is a self-hosted meeting booking system built as a single Go binary. It syncs iCal calendars, computes available slots, and exposes a REST API consumed by a Hugo blog widget. All code lives in `backend/`.

## Build & Run

```bash
cd backend

make build          # produces ./timeslot binary
make run            # go run . -config=config.json
make tidy           # go mod tidy
make clean          # remove binary and timeslot.db

go test ./...                                              # run all tests
go test ./integration/...                                  # run integration tests
go test -run TestCreateBooking_Conflict ./integration/...  # run a single test
```

Copy `config.example.json` → `config.json` before running.

## Architecture

```
main.go
  ├── config.Load()          – JSON config with defaults
  ├── db.Open()              – SQLite, WAL mode, runs migrations
  ├── models.*Store          – one struct per table, injected everywhere
  ├── engine.Engine          – slot computation (core logic)
  ├── sync.Worker            – iCal fetcher, runs every 15m via cron
  ├── email.Mailer           – SMTP notifications
  ├── api.Handler            – public REST API (/api/*)
  └── admin.Handler          – admin UI + JSON endpoints (/admin/*)
```

The **engine** is the heart of the system: `ComputeSlots()` expands `availability_rules`, subtracts `busy_blocks` (from calendar sync) and confirmed `bookings`, then slices the remainder into fixed-duration slots.

## Package Conventions

### Store Pattern
Every table has a dedicated `*Store` struct holding `*sql.DB`:
```go
type BookingStore struct{ DB *sql.DB }
func (s *BookingStore) List() ([]Booking, error) { ... }
func (s *BookingStore) Get(id int64) (*Booking, error) { ... }
func (s *BookingStore) Create(b *Booking) error { ... }
```
Stores are constructed in `main.go` and injected into handlers — no globals, no ORM.

### HTTP Handlers
- No router framework; uses stdlib `net/http.ServeMux`.
- Middleware is a simple higher-order function: `withCORS(h)`, `auth(h, user, pass)`.
- Admin routes use path-based dispatch (string matching on `/admin/` prefix) inside a single handler.
- Admin is protected by HTTP Basic Auth (`AdminUser`/`AdminPassword` from config).
- Public API routes have `withCORS` applied.
- Responses: `http.Error()` for errors; 201 for booking creation, 409 for conflicts, 303 redirects from admin forms.

### Error Handling
- Errors are logged with `log.Printf()` and returned as `http.Error()` — no panic-based error flow.
- Email notifications are sent in goroutines (fire-and-forget) to avoid blocking the booking response.

### Config
Loaded from a JSON file (default `config.json`). Key fields:
- `slot_duration_min`, `buffer_before_min`, `max_days_ahead` — slot computation tuning
- `auto_confirm` — skip manual approval flow
- `timezone` — IANA name; availability rules are interpreted in this zone
- `encryption_key` — 32 hex chars (16-byte AES) for calendar credentials
- `secret_key` — HMAC key used for token signing

### Templates
HTML templates live in `backend/templates/*.html` and are parsed at startup with `html/template`. The binary must be run from the `backend/` directory (or templates path adjusted) so the glob resolves correctly.

### Database
SQLite with WAL mode and foreign keys enabled. Schema migrations run automatically in `db.Open()`. Nullable timestamps use `sql.NullTime`.

## Key Dependencies

| Package | Purpose |
|---|---|
| `github.com/arran4/golang-ical` | iCal parsing |
| `github.com/mattn/go-sqlite3` | SQLite (CGo) |
| `github.com/robfig/cron/v3` | 15-minute sync schedule |
