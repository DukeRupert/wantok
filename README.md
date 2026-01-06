# Wantok

A simple, self-hosted instant messaging application for family communication.

## Overview

Wantok is a lightweight, web-based messaging platform designed for small family groups. It provides direct messaging between family members with real-time delivery, accessible from any device with a web browser.

The name comes from Tok Pisin (Papua New Guinea Pidgin), where "wantok" means "one talk"—someone who speaks your language. It's used to refer to clan members, extended family, and close community.

## Features

- **Direct messaging** between any two family members
- **Real-time delivery** via WebSocket
- **Multi-device support** — same account works on phone, tablet, and desktop simultaneously
- **30-day message history** with automatic cleanup
- **Admin user management** — invite-only, no self-registration
- **Cross-platform** — works on Android, iOS, macOS, Linux, Windows via web browser

## Non-Features (Intentional)

- No group chats (keeps it simple)
- No file/image uploads (text only)
- No read receipts or typing indicators
- No push notifications (in-app only)
- No message deletion (once sent, it stays until 30-day expiry)

## Technology Stack

### Backend

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | Go | Single binary deployment, excellent concurrency for WebSockets, strong standard library |
| Database | SQLite | Zero configuration, single file, sufficient for family-scale usage, easy backup |
| Router | chi or net/http | Minimal dependencies, chi adds method routing if using Go <1.22 |
| WebSocket | gorilla/websocket or nhooyr.io/websocket | Mature, well-tested WebSocket implementations |
| Password hashing | golang.org/x/crypto/bcrypt | Industry standard, built-in work factor |
| SQLite driver | modernc.org/sqlite | Pure Go, no CGO required for easier cross-compilation |

### Frontend

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Templating | html/template (stdlib) | Server-rendered, no build step |
| Interactivity | HTMX + Alpine.js | Minimal JavaScript, progressive enhancement |
| Styling | Tailwind CSS | Utility-first, easy responsive design |
| Real-time | Native WebSocket API | No additional libraries needed |

### Infrastructure

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Hosting | Self-hosted VPS | Full control, privacy |
| Reverse proxy | Caddy | Automatic HTTPS, simple WebSocket proxying |
| Process manager | systemd | Standard on Linux, handles restarts |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Clients                             │
│  (Browser on phone, tablet, laptop - any OS)                │
└─────────────────┬───────────────────────────┬───────────────┘
                  │ HTTPS                     │ WSS
                  ▼                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    Caddy (reverse proxy)                    │
│                    - TLS termination                        │
│                    - WebSocket proxying                     │
└─────────────────┬───────────────────────────┬───────────────┘
                  │ HTTP                      │ WS
                  ▼                           ▼
┌─────────────────────────────────────────────────────────────┐
│                     Go Application                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   Handlers  │  │  Middleware │  │    WebSocket Hub    │  │
│  │  (REST API) │  │   (Auth)    │  │ (real-time delivery)│  │
│  └──────┬──────┘  └─────────────┘  └──────────┬──────────┘  │
│         │                                     │              │
│         ▼                                     │              │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                    Models / Repository                  ││
│  └─────────────────────────┬───────────────────────────────┘│
└────────────────────────────┼────────────────────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │     SQLite      │
                    │   (WAL mode)    │
                    └─────────────────┘
```

## Data Model

Three tables:

**users** — family member accounts
- id, username, display_name, password_hash, is_admin, created_at

**sessions** — authentication tokens
- token, user_id, created_at, expires_at (30 days)

**messages** — conversation history
- id, sender_id, recipient_id, content, created_at

Conversations are implicit — derived from message pairs between two users.

## Project Structure

```
wantok/
├── cmd/
│   └── server/
│       └── main.go              # Entry point, config, wiring
├── internal/
│   ├── auth/
│   │   ├── password.go          # bcrypt helpers
│   │   ├── session.go           # Token management
│   │   └── middleware.go        # RequireAuth, RequireAdmin
│   ├── database/
│   │   ├── db.go                # Connection, WAL mode
│   │   └── migrations.go        # Schema versioning
│   ├── models/
│   │   ├── user.go
│   │   ├── session.go
│   │   └── message.go
│   ├── handlers/
│   │   ├── auth.go
│   │   ├── users.go
│   │   ├── admin.go
│   │   ├── messages.go
│   │   └── websocket.go
│   └── realtime/
│       ├── hub.go               # Connection registry, broadcast
│       └── client.go            # Per-connection goroutines
├── web/
│   ├── templates/
│   │   ├── base.html
│   │   ├── login.html
│   │   ├── chat.html
│   │   └── admin.html
│   └── static/
│       └── app.css
├── docs/
│   └── IMPLEMENTATION.md        # Detailed build plan
├── go.mod
└── go.sum
```

## Getting Started

### Prerequisites

- Go 1.22+
- A VPS with a domain name (for production)

### Development

```bash
# Clone and enter directory
git clone <repo>
cd wantok

# Create first admin user
go run ./cmd/server --create-admin

# Start server
go run ./cmd/server

# Visit http://localhost:8080
```

### Production Deployment

See deployment section in [IMPLEMENTATION.md](docs/IMPLEMENTATION.md#deployment-notes).

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `WANTOK_LISTEN_ADDR` | `:8080` | Address to bind |
| `WANTOK_DATABASE_PATH` | `./wantok.db` | SQLite file location |
| `WANTOK_SESSION_MAX_AGE` | `2592000` | Session lifetime in seconds (30 days) |
| `WANTOK_SECURE_COOKIES` | `false` | Set `true` in production with HTTPS |

## License

All rights reserved.