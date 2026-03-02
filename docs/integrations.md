# HomeLab Integrations (Zitadel & Gotify)

> **Status**: These integrations are currently **not implemented** and are listed here for future consideration. The current Timeslot release is a minimal, self-contained system.

This document outlines the design for integrating Timeslot with common HomeLab services: **Zitadel** (for Single Sign-On) and **Gotify** (for instant push notifications).

---

## 1. Zitadel Integration (SSO)

Currently, the Admin UI uses simple HTTP Basic Auth. Integrating Zitadel allows for a more secure, centralized authentication experience via OpenID Connect (OIDC).

### Design
- **Protocol**: OIDC (OpenID Connect).
- **Flow**: Authorization Code Flow with PKCE.
- **Backend Changes**:
  - Replace or supplement the `h.auth` middleware in `backend/admin/handler.go`.
  - Use an OIDC client library (e.g., `coreos/go-oidc`) to handle the redirection and token validation.
  - Store session information in a secure, HTTP-only cookie.
- **Zitadel Setup**:
  - Create a new Project and Application (Web) in Zitadel.
  - Configure Redirect URIs (e.g., `https://timeslot.example.com/admin/login/callback`).

### Proposed Configuration (`config.json`)
```json
{
  "oidc": {
    "enabled": true,
    "issuer": "https://zitadel.example.com",
    "client_id": "your-client-id",
    "client_secret": "your-client-secret",
    "redirect_url": "https://timeslot.example.com/admin/login/callback"
  }
}
```

---

## 2. Gotify Integration (Push Notifications)

Gotify provides a simple API to send push notifications to mobile devices or desktop clients. This serves as the primary notification mechanism for the owner for system events (e.g. sync failures).

### Design
- **Mechanism**: Send an HTTP POST request to the Gotify server when certain events occur.
- **Integration Point**: Add Gotify support to the `Notifier` interface.
- **Content**: Event details and a direct link to the Admin UI.

### Proposed Configuration (`config.json`)
```json
{
  "gotify": {
    "enabled": true,
    "url": "https://gotify.example.com",
    "token": "your-app-token",
    "priority": 5
  }
}
```

### Backend Logic
```go
func (n *Notifier) SendGotify(title, message string) error {
    if !cfg.Gotify.Enabled { return nil }
    // POST to {cfg.Gotify.URL}/message?token={cfg.Gotify.Token}
    // Body: {"title": title, "message": message, "priority": cfg.Gotify.Priority}
}
```

---

## Implementation Roadmap

| Feature | Description | Status |
|---|---|---|
| **Gotify Support** | Add Gotify as the primary notification channel for owner alerts. | Not Implemented |
| **OIDC Support** | Implement OIDC login flow using Zitadel. | Not Implemented |
| **Role Mapping** | Map Zitadel roles/groups to Timeslot admin permissions. | Not Implemented |

---

## Benefits
- **Zitadel**: Multi-factor authentication (MFA), centralized user management, and no more manual `admin_password` management in config files.
- **Gotify**: Instant alerts, free self-hosted push notifications, and zero reliance on external email services.
