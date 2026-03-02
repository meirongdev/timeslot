# Security Design

Timeslot is a read-only visibility system for visitors, which naturally reduces its attack surface. Security efforts are focused on protecting the Admin UI.

## 1. Admin UI Protection

- **HTTP Basic Auth**: Access to `/admin/*` is protected by `admin_user` and `admin_password` defined in `config.json`.
- **Recommendation**: Deploy Timeslot behind a proxy (like Cloudflare or Nginx) that implements rate limiting or fail2ban for the Admin UI.

## 2. Booking Confirmation

- **Manual Confirmation**: Booking requests are handled manually by the owner. The system does not have automatic booking confirmation.
- **Workflow**: Visitors see available slots via the API, then contact the owner through external channels. The owner confirms by adding the event to their own Google Calendar.

## 3. Public API

- **Read-Only**: The public API (`/api/slots`, `/api/schedule`) is read-only and exposes no sensitive data.
- **Recommendation**: Use external rate limiting (e.g., Cloudflare WAF) if needed to prevent abuse.

## 4. Future Enhancements (Not Implemented)

| Feature | Description | Status |
|---|---|---|
| Internal Rate Limiting | Middleware to limit requests per IP | Not Implemented |
| Honeypot Protection | Anti-spam measures for booking forms | Not Implemented |
| Turnstile/CAPTCHA | Bot protection for public endpoints | Not Implemented |
