# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Wantok is a self-hosted family instant messaging application built with Go. It provides direct messaging between family members with real-time WebSocket delivery. The name comes from Tok Pisin meaning "one talk" (clan/family members).

Key constraints: No group chats, no file uploads, no push notifications, text-only messages with 30-day expiry.

## Build and Run Commands

```bash
# Run the server (development)
go run ./cmd/server

# Build binary (pure Go, no CGO)
CGO_ENABLED=0 go build -o wantok ./cmd/server

# Generate SQLC code after modifying queries
sqlc generate

# Create new migration
goose -dir internal/database/migrations create <name> sql
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_PATH` | `wantok.db` | SQLite file path |
| `PORT` | `8080` | Server port |
| `SESSION_SECRET` | `PaxRomana` | Session signing secret |
| `SESSION_MAX_AGE` | `3600` | Session lifetime in seconds |
| `SECURE_COOKIES` | `true` | Set to `false` for local HTTP development |

## Architecture

### Code Organization

```
cmd/server/main.go     - Entry point, config loading, HTTP server setup
internal/
  auth/                - Password hashing (bcrypt), session management, middleware
  cleanup/             - Background job for expired sessions and old messages
  database/            - SQLite connection with embedded goose migrations
  handlers/            - HTTP handlers (auth, users, admin, messages, websocket)
  realtime/            - WebSocket hub and client management for real-time delivery
  render/              - HTML template rendering
  store/               - SQLC-generated database queries
  validate/            - Input validation for usernames, passwords, messages
deploy/
  wantok.service       - Systemd service file
  Caddyfile            - Caddy reverse proxy config
  backup.sh            - Database backup script
```

### Key Patterns

**Database Layer**: Uses SQLC for type-safe SQL queries. Migrations are embedded via `//go:embed` and run automatically on startup with goose. SQLite configured with WAL mode, foreign keys enabled, 5s busy timeout.

**HTTP Server**: Standard library `net/http` with `http.ServeMux`. Handlers receive `*store.Queries` for database access. Auth middleware extracts user from session cookie and stores in request context.

**Real-time**: Hub pattern for WebSocket connections - single goroutine manages client map. Messages sent via REST API trigger WebSocket delivery to both sender's and recipient's connected clients.

### Data Flow

1. REST API handles message creation (POST /conversations/:userID/messages)
2. Message persisted to SQLite via SQLC query
3. Hub broadcasts to recipient's and sender's WebSocket clients
4. Frontend receives via WebSocket and updates UI

### SQLC Usage

- Queries: `internal/database/queries/*.sql`
- Migrations: `internal/database/migrations/*.sql`
- Generated code: `internal/store/`
- Config: `sqlc.yaml`

Run `sqlc generate` after modifying any `.sql` query files.

## Implementation Status

See `dev/checklist.md` for detailed progress tracking. Completed phases:
- Phase 1: Auth Foundation (login, logout, sessions, middleware)
- Phase 2: User Management (admin CLI, user CRUD)
- Phase 3: Messaging (conversations, message sending via REST)
- Phase 4: Real-Time Delivery (WebSocket hub, live message updates)
- Phase 5: Cleanup and Hardening (background jobs, validation, mobile UI)

## Security Notes

- CSRF protection via SameSite=Lax cookies (form submissions require same-origin)
- XSS prevention via Go's html/template automatic escaping
- SQL injection prevention via SQLC parameterized queries
- Session cookies are HttpOnly and Secure (in production)
- Input validation on all user-supplied data (username, password, messages)
