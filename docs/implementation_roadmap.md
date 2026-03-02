# Timeslot — Implementation Overview

## What It Is

Timeslot is a minimal, self-hosted calendar visibility system.  
It is a single Go binary backed by SQLite. No heavy frameworks, no cloud services required.

---

## Core Features

### 1. iCal Calendar Sync

- Add iCal feed URLs (Google Calendar, Outlook, etc.) in the Admin UI.
- A background worker syncs each calendar every 15 minutes.
- Synced events are stored as **busy blocks** (title retained for admin only).

### 2. Availability Rules

- Define weekly recurring availability windows (e.g. Mon–Fri, 09:00–17:00) in the Admin UI.
- Slots are computed by subtracting busy blocks from these windows.

### 3. Public REST API

| Endpoint | Description |
|---|---|
| `GET /api/slots` | Returns available time slots |
| `GET /api/schedule` | Returns all slot windows with `available` / `occupied` status (no event details leaked) |

All public endpoints have CORS headers for embedding in static sites (Hugo, etc.).

### 4. Admin UI

- **Calendars** — add, delete, manually sync iCal feeds.
- **Availability** — manage weekly availability rules.

Protected by HTTP Basic Auth (`admin_user` / `admin_password` in config).

---

## Configuration

Minimal `config.json`:

```json
{
  "listen_addr": ":8080",
  "db_path":     "./timeslot.db",
  "admin_user":     "admin",
  "admin_password": "changeme",
  "slot_duration_min": 30,
  "buffer_before_min": 0,
  "max_days_ahead":    30,
  "timezone": "Asia/Shanghai"
}
```

---

## What Was Deliberately Excluded

The following were considered and removed to keep the system simple:

| Feature | Reason removed |
|---|---|
| Booking Requests | Out of scope; system only provides visibility |
| Admin confirm/cancel buttons | Visibility is driven by iCal sync from owner's primary calendar |
| Email notifications | No booking flow means no notification system needed |
| ICS attachment | Not needed for visibility-only system |
| Secret/encryption keys | No guest tokens or encrypted credentials storage |
| OIDC / SSO | Admin UI uses Basic Auth — sufficient for personal use |
| OpenTelemetry | Not needed at this scale |
| Gotify push notifications | Considered an optional integration; not in core |
| Rate limiting / honeypot / Turnstile | Public endpoints are read-only; minimal spam risk |
