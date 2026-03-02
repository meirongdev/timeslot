# Timeslot

A self-hosted calendar visibility system. Syncs your iCal calendars, computes available slots, and exposes a REST API for a status widget (e.g., embedded in a Hugo blog).

## Features

- Sync multiple iCal URL calendars on a 15-minute schedule
- Configurable weekly availability rules with date bounds
- Public REST API for querying available slots and schedule
- Admin UI (server-side rendered) for managing calendars and availability
- Single Go binary, SQLite storage — minimal footprint (~20 MB RAM)

## Quick Start

```bash
cd backend
cp config.example.json config.json
# Edit config.json with your settings
make run
```

The admin UI is at `http://localhost:8080/admin/` (default credentials: `admin` / `changeme`).

## Requirements

To get Timeslot running, you will need:

1.  **iCal URL**: A secret iCal (`.ics`) link from your calendar provider (e.g., Google, Apple, or Outlook).
2.  **Admin Password**: A strong password for the management dashboard.

See [docs/configuration.md](docs/configuration.md) for a detailed "Where to get this information?" guide.

## Configuration

Copy `backend/config.example.json` and edit:

| Field               | Default         | Description                                                              |
| ------------------- | --------------- | ------------------------------------------------------------------------ |
| `listen_addr`       | `:8080`         | HTTP bind address                                                        |
| `db_path`           | `./timeslot.db` | SQLite database file                                                     |
| `admin_user`        | `admin`         | Admin UI username                                                        |
| `admin_password`    | `changeme`      | Admin UI password — **change this**                                      |
| `slot_duration_min` | `30`            | Duration of each bookable slot (minutes)                                 |
| `buffer_before_min` | `0`             | Minimum lead time before a slot can be booked (minutes)                  |
| `max_days_ahead`    | `30`            | How far in the future slots are shown                                    |
| `timezone`          | `UTC`           | IANA timezone for interpreting availability rules (e.g. `Asia/Shanghai`) |

See [docs/configuration.md](docs/configuration.md) for details.

## API

```
GET  /api/slots?from=<RFC3339>&to=<RFC3339>      # available slots
GET  /api/schedule?from=<RFC3339>&to=<RFC3339>   # privacy-first schedule
```

See [docs/api.md](docs/api.md) for full request/response documentation.

## Deployment

See [docs/deployment.md](docs/deployment.md) for Docker and Kubernetes deployment guides.

## Build

```bash
cd backend
make build   # produces ./timeslot
make tidy    # go mod tidy
make clean   # remove binary and timeslot.db
```

Requires Go 1.25+ and CGo (for `go-sqlite3`). On Linux, `gcc` must be available.

## Architecture

```
main.go
  ├── config     – JSON config loader
  ├── db         – SQLite open + auto-migration
  ├── models     – Store pattern (CalendarStore, BusyBlockStore, AvailabilityStore)
  ├── engine     – Slot computation (availability rules − busy blocks)
  ├── sync       – iCal fetch worker (runs every 15 min)
  ├── api        – Public REST API (/api/*)
  └── admin      – Admin UI (/admin/*)
```

## Known Limitations

- Only iCal URL calendars are supported. CalDAV and Google Calendar direct integration are not yet implemented.
