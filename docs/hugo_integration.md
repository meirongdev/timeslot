# Hugo Integration — Timeslot widget

This document describes how Timeslot is integrated into the Hugo blog (Meirong.dev) so future maintainers can understand and reproduce the setup.

## Overview

- Public Timeslot base URL used: `https://slot.meirong.dev`
- The blog shows visitor-facing available time slots (next 14 days) fetched from Timeslot's `/api/slots` endpoint.
- The page is read-only: visitors are instructed to email `meirongdev@gmail.com` to request a meeting (no inline booking forms).

## Files changed / created (blog repository)

- `layouts/shortcodes/rawhtml.html`
  - Content: `{{- .Inner | safeHTML -}}`
  - Purpose: allow embedding raw HTML/JS blocks inside Markdown pages safely.

- `content/timeslot.md`
  - New page that mounts the Timeslot widget on `/timeslot`.
  - Fetches `https://slot.meirong.dev/api/slots?from=...&to=...` and renders slots grouped by date.
  - Displays the next 14 days of available slots and provides a `mailto:meirongdev@gmail.com?subject=预约申请` link for visitors.

- `config.toml`
  - Added a menu entry under `languages.en.menu.main`:

```toml
[[languages.en.menu.main]]
identifier = "timeslot"
weight = 30
name = "Timeslot"
url = "/timeslot"
```

(About/other menu weights were shifted accordingly.)

## Widget behavior (implementation notes)

- The widget requests `/api/slots` with RFC3339 `from` and `to` query params (now → now+14d).
- The widget expects a JSON array of slots:

```json
[ {"start":"2026-03-03T09:00:00Z","end":"2026-03-03T09:30:00Z"}, ... ]
```

- It groups slots by date (client timezone `Asia/Shanghai` by default) and renders each slot with a time range and an "空闲" badge.
- No forms or POSTs are performed by the widget. Visitors must email `meirongdev@gmail.com` to request a meeting.

## Shortcode / embedding

- The site uses `{{< rawhtml >}} ... {{< /rawhtml >}}` in `content/timeslot.md` to embed the widget's HTML+CSS+JS. The `rawhtml` shortcode simply returns `{{- .Inner | safeHTML -}}`.

## API & CORS

- Ensure Timeslot exposes CORS headers for the widget to fetch the slots from the blog origin, e.g.:
  - `Access-Control-Allow-Origin: *`
  - `Access-Control-Allow-Methods: GET, OPTIONS`
  - `Access-Control-Allow-Headers: Content-Type`

- Primary endpoint used: `https://slot.meirong.dev/api/slots` (client-side fetch). If you prefer a privacy-first masked schedule, `/api/schedule` can be used instead.

## Testing & verification

1. Run Hugo locally and open the widget:

```bash
cd /path/to/blog
hugo server -D
# open http://localhost:1313/timeslot
```

2. In browser DevTools -> Network, verify:
   - GET `https://slot.meirong.dev/api/slots?from=...&to=...` returns `200 OK` and JSON array.
   - No CORS errors in the console.

3. Check date/time formatting: the widget uses `Asia/Shanghai` by default. If you want a different display TZ, edit the `TZ` constant in the widget JS inside `content/timeslot.md`.

4. If no slots appear, curl the endpoint directly to ensure Timeslot service responds:

```bash
curl 'https://slot.meirong.dev/api/slots?from=$(date --utc +%Y-%m-%dT%H:%M:%SZ)&to=$(date --utc -d "+14 days" +%Y-%m-%dT%H:%M:%SZ)' | jq .
```

(Adjust `date` syntax for macOS: `date -v+14d -u +%Y-%m-%dT%H:%M:%SZ`.)

## Troubleshooting

- CORS errors: enable appropriate CORS headers on the Timeslot service.
- Network errors: ensure `slot.meirong.dev` is reachable from client browsers (DNS / SSL).
- Empty results: check Timeslot sync with upstream calendars and its `max_days_ahead` / `slot_duration_min` settings.

## Optional improvements

- Add a toggle on the page to switch between `/api/slots` and `/api/schedule` (masked schedule).
- Optionally implement an owner-reviewed booking form that POSTs to Timeslot's `POST /api/bookings` (if the Timeslot service exposes it).
- Localize the widget timezone dynamically from site config if multiple locales are supported.

## Commit & deploy suggestions

```bash
# from blog repo
git add layouts/shortcodes/rawhtml.html content/timeslot.md config.toml
git commit -m "Add Timeslot read-only widget page and menu entry"
git push origin main
```

After pushing, verify the production site page `/timeslot` loads and that the widget fetches `https://slot.meirong.dev/api/slots` successfully.

---

Document created to capture the current Timeslot ↔ Hugo integration (read-only widget). Update this file if you change the endpoint, display timezone, or page location.
