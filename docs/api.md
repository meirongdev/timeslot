# API Reference

All public endpoints are prefixed with `/api/` and support CORS with the following headers:
- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, POST, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type`

Timestamps use **RFC3339** format (e.g., `2024-05-01T09:00:00Z`).

---

## GET /api/slots

Returns available time slots in the requested range based on availability rules and busy blocks.

### Query Parameters

| Parameter | Required | Description |
|---|---|---|
| `from` | No | Start of range (RFC3339). Defaults to now. |
| `to` | No | End of range (RFC3339). Defaults to 7 days from now. Capped at `max_days_ahead`. |

### Response `200 OK`

```json
[
  {
    "start": "2024-05-01T09:00:00Z",
    "end":   "2024-05-01T09:30:00Z"
  },
  {
    "start": "2024-05-01T10:00:00Z",
    "end":   "2024-05-01T10:30:00Z"
  }
]
```

Returns `[]` if no slots are available. Slots respect `buffer_before_min`.

---

## GET /api/schedule

Returns a privacy-first schedule: each entry is either "available" or "occupied", with no event details exposed.

### Query Parameters

Same as `/api/slots`.

### Response `200 OK`

```json
[
  {
    "start": "2024-05-01T09:00:00Z",
    "end":   "2024-05-01T09:30:00Z",
    "status": "available"
  },
  {
    "start": "2024-05-01T09:30:00Z",
    "end":   "2024-05-01T10:00:00Z",
    "status": "occupied"
  }
]
```

---

## POST /api/bookings (Planned)

This endpoint is planned for future releases to allow visitors to submit booking requests directly through the API.

### Request (Future)

```json
{
  "slot_start": "2024-05-01T09:00:00Z",
  "slot_end": "2024-05-01T09:30:00Z",
  "visitor_name": "John Doe",
  "visitor_email": "john@example.com",
  "notes": "Appointment for consultation"
}
```

### Response (Future)

```json
{
  "id": "uuid",
  "status": "pending",
  "message": "Booking request received. The host will confirm via Google Calendar."
}
```

### Current Workflow

Currently, booking requests are handled manually:
1. Visitors see available slots via `/api/slots`
2. They contact the owner through external channels (email, form, etc.)
3. The owner confirms by adding the event to their Google Calendar

Admin routes require **HTTP Basic Auth** (configured via `admin_user` / `admin_password`).

| Method | Path | Description |
|---|---|---|
| GET | `/admin/` | Dashboard |
| GET/POST | `/admin/calendars` | List / add iCal calendars |
| GET | `/admin/calendars/sync/:id` | Trigger manual sync |
| GET | `/admin/calendars/delete/:id` | Delete calendar |
| GET/POST | `/admin/availability` | List / add availability rules |
| GET | `/admin/availability/delete/:id` | Delete rule |

---

## Hugo Widget Integration

Timeslot can be embedded in any static site (Hugo, Jekyll, Next.js, etc.) via the public API. CORS is enabled by default.

### Quick Start

1. Copy the sample file to your Hugo project:
   ```bash
   cp docs/hugo_sample.html layouts/partials/timeslot.html
   ```

2. Add to your template:
   ```html
   {{ partial "timeslot.html" . }}
   ```

3. Update `API_BASE` in the HTML to point to your Timeslot instance.

### Using as a Shortcode

Create `layouts/shortcodes/timeslot.html`:

```html
<div class="timeslot-widget" data-api="{{ .Get "api" | default "schedule" }}">
  <!-- Widget renders here -->
</div>
<script src="/js/timeslot.js"></script>
```

Usage: `{{< timeslot api="schedule" >}}`

### Customizing the Widget

Edit `API_BASE` and `TIMEZONE` in the JavaScript section:

```javascript
const API_BASE = "https://timeslot.example.com/api";
const TIMEZONE = "Asia/Singapore"; // Change to your desired timezone
```

### Display Modes

| Endpoint | Use Case |
|---|---|
| `/api/slots` | Show only available slots (good for booking forms) |
| `/api/schedule` | Show availability/occupancy (good for visibility) |

### Example: Fetch Available Slots

```javascript
const from = new Date().toISOString();
const to = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString();

fetch(`${API_BASE}/slots?from=${from}&to=${to}`)
  .then(r => r.json())
  .then(slots => {
    slots.forEach(slot => {
      console.log(`Available: ${slot.start} - ${slot.end}`);
    });
  });
```

### Security Notes

- The public API is read-only and exposes no sensitive data.
- If needed, restrict API access by IP using a reverse proxy (nginx, Cloudflare).
- The widget runs entirely client-side; no server-side rendering required.
