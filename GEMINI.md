# Timeslot - Project Context

Timeslot is a self-hosted calendar visibility system designed for personal use (e.g., embedded in a static blog like Hugo). It synchronizes with external iCal calendars, calculates availability based on rules, and provides a REST API for availability widgets and an Admin UI for management.

## Project Overview

- **Core Functionality**:
  - Syncs external iCal URL calendars every 15 minutes.
  - Computes available slots by subtracting busy blocks (from sync) from user-defined availability rules.
  - Exposes a public REST API for querying slots and schedule status.
  - Provides a password-protected Admin UI for managing calendars and availability rules.
- **Tech Stack**:
  - **Language**: Go 1.25+
  - **Database**: SQLite (local file storage)
  - **Frontend**: Server-side rendered HTML templates (Go `html/template`).
  - **Deployment**: Docker, Kubernetes (Helm charts included).
- **Key Modules**:
  - `backend/config`: JSON configuration loader.
  - `backend/db`: SQLite connection and automatic schema migration.
  - `backend/models`: Data access layer (Calendars, BusyBlocks, Availability).
  - `backend/engine`: Core logic for computing available time slots.
  - `backend/sync`: Background worker for iCal synchronization.
  - `backend/api`: Public-facing REST API.
  - `backend/admin`: Admin dashboard for management.

## Building and Running

Commands should be executed from the `backend/` directory.

- **Build**: `make build` (produces `timeslot` binary).
- **Run**: `make run` (runs the server; requires `config.json`).
- **Development Run**: `go run . -config=config.json`
- **Tidy Dependencies**: `make tidy`
- **Clean**: `make clean` (removes binary and local `timeslot.db`).

### Setup

1. `cd backend`
2. `cp config.example.json config.json`
3. Edit `config.json` (ensure `admin_password` is set).
4. Run `make run`.

## Development Conventions

- **Architecture**: Follows a modular approach with clear separation between the data layer (`models`), business logic (`engine`), and transport layers (`api`, `admin`).
- **Database**: Uses a single SQLite file. Schema changes should be added to the `schema` constant in `backend/db/migrate.go`.
- **Templates**: Admin UI templates are located in `backend/templates/`. The server parses them at startup using `ParseGlob`.
- **Configuration**: Managed via `config.json`. Defaults are provided in `backend/config/config.go`.
- **Testing**: Integration tests are located in `backend/integration/`. Use `go test ./...` to run tests.
- **Security**: Access protection strategies are documented in `docs/security.md`.
- **Integrations**: Support for HomeLab services like Zitadel (SSO) and Gotify (Notifications) is documented in `docs/integrations.md`.
- **Observability**: OpenTelemetry (OTel) integration for tracing and metrics is documented in `docs/observability.md`.
- **Hugo Integration**: The Privacy-First design for Hugo blogs (Available vs. Occupied) is documented in `docs/hugo_integration.md`.
- **Hugo Sample**: A complete HTML/JS/CSS integration sample is available in `docs/hugo_sample.html`.
- **Roadmap**: The overall implementation plan and feature tracking are documented in `docs/implementation_roadmap.md`.
- **Styling**: The Admin UI uses simple CSS (embedded or minimal) within the templates.

## API Endpoints

- `GET  /api/slots?from=<RFC3339>&to=<RFC3339>`: List available slots.
- `GET  /api/schedule?from=<RFC3339>&to=<RFC3339>`: List schedule status (available/occupied).

The Admin UI is accessible at `/admin/` (default credentials: `admin` / `changeme`).
