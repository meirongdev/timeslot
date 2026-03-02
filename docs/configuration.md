# Configuration Reference

Timeslot is configured via a JSON file passed with `-config <path>` (default: `config.json`).

Start from the example:

```bash
cp backend/config.example.json config.json
```

## Full Field Reference

```jsonc
{
  // HTTP server
  "listen_addr": ":8080",         // bind address; use "0.0.0.0:8080" in containers

  // Database
  "db_path": "/data/timeslot.db", // SQLite file; use a persistent volume in k8s

  // Admin UI credentials
  "admin_user":     "admin",
  "admin_password": "changeme",   // CHANGE IN PRODUCTION

  // Slot behaviour
  "slot_duration_min": 30,    // each slot length in minutes
  "buffer_before_min": 0,   // minimum lead time (minutes) before a slot is shown as available
  "max_days_ahead":    30,    // how many days ahead to expose availability

  // Timezone
  "timezone": "Asia/Shanghai"  // IANA name; availability rules are interpreted in this zone
}
```

## Notes

### Timezone
The `timezone` field affects how availability rules (weekday + start/end time) are expanded into actual time windows. All times stored in the database are UTC.

### Admin Credentials
Change `admin_password` before deploying to production. The Admin UI is accessible at `/admin/` and uses HTTP Basic Auth.

## Where to Get This Information?

### 1. iCal URLs (from your calendar provider)
These are added in the Admin UI (`/admin/calendars`).
*   **Google Calendar:** Go to Settings > [Your Calendar] > Integrate calendar > Secret address in iCal format.
*   **Apple iCloud:** Share calendar > Public Calendar (Note: Timeslot supports secret tokens in URLs).
*   **Outlook:** Settings > View all Outlook settings > Calendar > Shared calendars > Publish a calendar > ICS link.
*   **Fastmail:** Settings > Calendars > [Your Calendar] > Share > Public/Private iCal URL.

### 2. Timezone
Use an IANA Timezone name like `America/New_York` or `Europe/London`. You can find a list on [Wikipedia](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones).

## Database in Kubernetes
Set `db_path` to a path on a `PersistentVolumeClaim`, e.g., `/data/timeslot.db`. See [deployment.md](deployment.md).

### Availability Rules
Rules are configured in the Admin UI (`/admin/availability`). Each rule specifies:
- **Weekday** (0=Sunday … 6=Saturday)
- **Start time / End time** in `HH:MM` format (in the configured timezone)

Slots are computed as: `availability_rules − busy_blocks`, sliced to `slot_duration_min`, excluding slots within `buffer_before_min` of now.
