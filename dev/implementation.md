# Implementation Plan

This document provides a detailed, phase-by-phase implementation plan for building Wantok. Each phase ends with a testable checkpoint.

## Table of Contents

- [Phase 1: Auth Foundation](#phase-1-auth-foundation)
- [Phase 2: User Management](#phase-2-user-management)
- [Phase 3: Messaging (REST)](#phase-3-messaging-rest)
- [Phase 4: Real-Time Delivery](#phase-4-real-time-delivery)
- [Phase 5: Cleanup and Hardening](#phase-5-cleanup-and-hardening)
- [Deployment Notes](#deployment-notes)

---

## Phase 1: Auth Foundation

### 1.1 Project Initialisation

**Create directory structure**

Set up the folder hierarchy from the project structure. Create empty `.go` files as placeholders to establish the package structure. Initialise the Go module.

**Dependencies**

```
go get modernc.org/sqlite
go get github.com/go-chi/chi/v5
go get golang.org/x/crypto/bcrypt
go get github.com/gorilla/websocket
```

**Configuration struct**

Define a config struct to hold runtime configuration:

- `DatabasePath` — path to SQLite file
- `ListenAddr` — e.g. `:8080`
- `SessionSecret` — used for secure cookie signing (optional but recommended)
- `SessionMaxAge` — 30 days in seconds

Load from environment variables with sensible defaults.

### 1.2 Database Layer

**Connection management**

Create a function that opens the SQLite database, enables WAL mode and foreign keys, and returns a `*sql.DB`. Store this in a struct or pass it explicitly to handlers.

**Migrations table**

Use a simple approach: a `schema_migrations` table with a single `version` integer. On startup, check current version and apply any migrations with a higher number.

**Migration 001: users table**

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    is_admin INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

**Migration 002: sessions table**

```sql
CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at TEXT NOT NULL
);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
```

**Checkpoint:** Application starts, creates database file, runs migrations.

### 1.3 Password Utilities

Location: `internal/auth/password.go`

Two functions:

**HashPassword(plain string) (string, error)**
- Use bcrypt with cost 12 (balance of security and speed)
- Return the hashed string

**CheckPassword(hash, plain string) bool**
- Use bcrypt's compare function
- Return true if match, false otherwise
- Constant-time comparison is handled by bcrypt

### 1.4 Session Management

Location: `internal/auth/session.go`

**Token generation**

Create a function that generates 32 random bytes and hex-encodes them (64 character string). Use `crypto/rand`.

**CreateSession(db, userID int64) (token string, err error)**
- Generate token
- Calculate expiry (now + 30 days)
- Insert into sessions table
- Return token

**ValidateSession(db, token string) (*User, error)**
- Query sessions joined with users
- Check `expires_at > now`
- Return the user struct if valid, nil/error if not

**DeleteSession(db, token string) error**
- Simple DELETE by token

**DeleteUserSessions(db, userID int64) error**
- For use when admin deletes a user or user changes password

### 1.5 Auth Middleware

Location: `internal/auth/middleware.go`

**RequireAuth middleware**
- Read session token from cookie (name: `session`)
- If missing, redirect to `/login`
- Validate token via `ValidateSession`
- If invalid/expired, clear cookie, redirect to `/login`
- Store user in request context for handlers to access

**RequireAdmin middleware**
- Call RequireAuth first
- Check `user.IsAdmin`
- If not admin, return 403 Forbidden

**Helper: GetUser(ctx context.Context) *User**
- Extract user from context
- Used by handlers

### 1.6 User Model

Location: `internal/models/user.go`

Define User struct matching the database schema.

Repository functions (take `*sql.DB` as first argument):

- `GetUserByID(db, id int64) (*User, error)`
- `GetUserByUsername(db, username string) (*User, error)`
- `ListUsers(db) ([]User, error)` — for conversation picker
- `CreateUser(db, username, displayName, passwordHash string, isAdmin bool) (int64, error)` — returns new ID
- `UpdateUser(db, id int64, displayName, passwordHash string, isAdmin bool) error`
- `DeleteUser(db, id int64) error`

### 1.7 Auth Handlers

Location: `internal/handlers/auth.go`

**GET /login**
- If already authenticated, redirect to `/`
- Render login template

**POST /auth/login**
- Parse form: username, password
- Look up user by username
- If not found or password doesn't match, re-render login with error
- Create session
- Set cookie: `session={token}`, HttpOnly, SameSite=Lax, Secure (if HTTPS), MaxAge=30 days, Path=/
- Redirect to `/`

**POST /auth/logout**
- Get token from cookie
- Delete session from database
- Clear cookie
- Redirect to `/login`

**GET /auth/me (optional, for HTMX or JS)**
- Return JSON with current user info
- Used by frontend to check auth state

### 1.8 Login Template

Location: `web/templates/login.html`

Simple form with:
- Username field
- Password field
- Submit button
- Error message area (conditionally rendered)

Use your base template layout. Keep styling minimal for now.

### 1.9 Main Entry Point

Location: `cmd/server/main.go`

Startup sequence:
1. Load configuration from environment
2. Open database connection
3. Run migrations
4. Create router
5. Register routes with handlers (inject db into handlers via closure or struct)
6. Start HTTP server

Route registration:
- Serve static files from `/static/`
- Mount auth handlers
- Protected routes wrapped with RequireAuth

### 1.10 Phase 1 Testing Checklist

- [ ] Server starts without error
- [ ] Database file created with correct tables
- [ ] Visiting `/` redirects to `/login`
- [ ] Login page renders
- [ ] Login with wrong credentials shows error
- [ ] Can't log in (no users exist yet)
- [ ] Manually insert a user via SQLite CLI, verify login works
- [ ] After login, cookie is set
- [ ] After login, visiting `/login` redirects away
- [ ] Logout clears cookie and redirects to `/login`
- [ ] Expired session (manually backdate in DB) forces re-login

---

## Phase 2: User Management

### 2.1 Admin Seeding via CLI

Location: `cmd/server/main.go` modifications

Add command-line flag: `--create-admin`

When flag is present:
1. Prompt for username (or accept as flag value)
2. Prompt for password (read from stdin with echo disabled, or accept as flag)
3. Hash password
4. Insert user with `is_admin = true`
5. Print success message
6. Exit (don't start server)

Handle case where username already exists (update password and set admin flag, or error).

### 2.2 Admin Handlers

Location: `internal/handlers/admin.go`

All routes require admin middleware.

**GET /admin**
- Fetch all users
- Render admin template with user list

**POST /admin/users**
- Parse form: username, display_name, password, is_admin (checkbox)
- Validate: username required, password required, reasonable lengths
- Hash password
- Create user
- Redirect to `/admin` with success flash (or just redirect)

**POST /admin/users/:id**
- Parse form: display_name, password (optional), is_admin
- If password provided, hash it; otherwise keep existing
- Update user
- Redirect to `/admin`

**POST /admin/users/:id/delete**
- Delete user
- Delete all their sessions (cascade should handle, but explicit is fine)
- Redirect to `/admin`

Note: Using POST for delete with a hidden form field or button is simpler than DELETE with JavaScript. HTMX can use `hx-delete` if you prefer.

### 2.3 Admin Template

Location: `web/templates/admin.html`

Two sections:

**Create user form**
- Username input
- Display name input
- Password input
- Admin checkbox
- Submit button

**User list table**
- Columns: username, display name, admin (yes/no), created date, actions
- Each row has:
  - Edit button (could expand inline form or link to edit page)
  - Delete button (with confirmation—use `onclick="return confirm(...)"` or HTMX confirm)

For simplicity, you can make edit work by clicking a row to populate the create form, then changing the form action dynamically. Or have a separate edit modal/page.

### 2.4 Users List Endpoint

Location: `internal/handlers/users.go`

**GET /users**
- Requires auth (not admin)
- Return JSON array of `{id, username, display_name}` for all users except the current user
- Used by chat UI to show who can be messaged

### 2.5 Phase 2 Testing Checklist

- [ ] `--create-admin` flag creates admin user
- [ ] Admin can log in
- [ ] Admin can access `/admin`
- [ ] Non-admin user (if one exists) gets 403 on `/admin`
- [ ] Admin can create new user
- [ ] New user appears in list
- [ ] Admin can edit user's display name
- [ ] Admin can change user's password (user can log in with new password)
- [ ] Admin can toggle admin flag
- [ ] Admin can delete user
- [ ] Deleted user's sessions are invalidated (they get logged out)
- [ ] `/users` returns list for authenticated user

---

## Phase 3: Messaging (REST)

### 3.1 Messages Migration

**Migration 003: messages table**

```sql
CREATE TABLE messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sender_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_messages_sender_id ON messages(sender_id);
CREATE INDEX idx_messages_recipient_id ON messages(recipient_id);
CREATE INDEX idx_messages_created_at ON messages(created_at);
```

### 3.2 Message Model

Location: `internal/models/message.go`

Message struct:
- ID, SenderID, RecipientID, Content, CreatedAt
- Optional: SenderDisplayName (populated via join for convenience)

Repository functions:

**CreateMessage(db, senderID, recipientID int64, content string) (*Message, error)**
- Insert and return the created message with ID and timestamp

**GetConversation(db, userA, userB int64, limit, offset int) ([]Message, error)**
- Select messages where (sender=A AND recipient=B) OR (sender=B AND recipient=A)
- Order by created_at DESC (newest first for pagination)
- Limit and offset for pagination
- Join with users to get sender display name

**GetConversationList(db, userID int64) ([]ConversationSummary, error)**
- Complex query: find all distinct users that userID has exchanged messages with
- For each, get the most recent message and timestamp
- Order by most recent first
- Return list of: `{UserID, DisplayName, LastMessage, LastMessageAt, IsLastMessageFromMe}`

### 3.3 Message Handlers

Location: `internal/handlers/messages.go`

**GET /conversations**
- Get current user from context
- Call GetConversationList
- Return JSON or render template partial (for HTMX)

**GET /conversations/:userID/messages**
- Validate :userID is a real user
- Parse pagination params: `?before=<timestamp>` or `?page=N`
- Get messages
- Return JSON array, oldest-first (reverse for display) or newest-first depending on UI needs

**POST /conversations/:userID/messages**
- Validate :userID exists and isn't current user
- Parse body: content (form or JSON)
- Validate content not empty, reasonable max length (e.g., 4000 chars)
- Create message
- Return created message as JSON (for HTMX or JS to append)

### 3.4 Chat UI

Location: `web/templates/chat.html`

**Layout**
- Left sidebar: conversation list (users you've messaged)
- Main area: message history with selected user
- Bottom: input field and send button

**Conversation list behaviour**
- On page load, fetch `/conversations`
- Clicking a conversation loads `/conversations/:id/messages` into main area
- Show display name and preview of last message

**Message area behaviour**
- Show messages in chronological order (oldest at top)
- Distinguish sent (right aligned) vs received (left aligned)
- Scroll to bottom on load

**Send message behaviour**
- On submit, POST to `/conversations/:id/messages`
- Append returned message to list
- Clear input field
- Scroll to bottom

**New conversation**
- Button to start new conversation
- Shows list of all users (from `/users`)
- Clicking one opens empty conversation

**Polling (temporary until WebSocket)**
- Every 3 seconds, fetch messages again if conversation is open
- Or: poll `/conversations` to update sidebar

### 3.5 Phase 3 Testing Checklist

- [ ] Messages table created
- [ ] User A can send message to User B
- [ ] User B sees message from User A
- [ ] Messages persist across page refresh
- [ ] Conversation list shows users with message history
- [ ] Conversation list ordered by most recent
- [ ] Can start new conversation with user who hasn't been messaged before
- [ ] Messages display in correct order
- [ ] Sent vs received messages visually distinguished
- [ ] Pagination works (create 50+ messages, verify older ones load)
- [ ] Cannot send message to non-existent user
- [ ] Cannot send empty message

---

## Phase 4: Real-Time Delivery

### 4.1 Hub Architecture

Location: `internal/realtime/hub.go`

The hub is a single goroutine that manages all active WebSocket connections.

**Data structures**

```go
type Hub struct {
    // Map of userID → slice of clients (multiple devices)
    clients map[int64][]*Client
    
    // Channels for goroutine communication
    register   chan *Client
    unregister chan *Client
    broadcast  chan *OutgoingMessage
}
```

**Hub methods**

- `NewHub() *Hub` — initialise maps and channels

- `Run()` — the main loop (run as goroutine):
  ```
  for {
      select {
      case client := <-h.register:
          // Add client to map under their userID
      case client := <-h.unregister:
          // Remove client from map, close their send channel
      case msg := <-h.broadcast:
          // Find recipient's clients, send to each
      }
  }
  ```

- `Register(client *Client)` — send client to register channel

- `Unregister(client *Client)` — send client to unregister channel

- `SendToUser(userID int64, msg OutgoingMessage)` — send to broadcast channel

### 4.2 Client Management

Location: `internal/realtime/client.go`

```go
type Client struct {
    hub    *Hub
    userID int64
    conn   *websocket.Conn
    send   chan []byte  // Buffered channel for outgoing messages
}
```

**ReadPump (goroutine per client)**
- Runs in a loop reading from WebSocket
- For this app, clients don't send meaningful messages (we use REST for sending)
- Just handle pings/pongs and detect disconnect
- On disconnect, call hub.Unregister and close connection

**WritePump (goroutine per client)**
- Runs in a loop reading from send channel
- Write each message to WebSocket
- Handle periodic ping writes to keep connection alive
- On send channel close, exit

**Message format**

```go
type OutgoingMessage struct {
    Type      string `json:"type"`      // "message"
    ID        int64  `json:"id"`
    From      int64  `json:"from"`
    FromName  string `json:"fromName"`
    Content   string `json:"content"`
    Timestamp string `json:"timestamp"`
}
```

### 4.3 WebSocket Handler

Location: `internal/handlers/websocket.go`

**GET /ws**
- Validate auth via cookie (WebSocket upgrade passes cookies)
- Upgrade connection to WebSocket
- Create Client struct with userID from auth
- Register client with hub
- Start ReadPump and WritePump goroutines

**Connection setup pseudocode**

```go
func handleWebSocket(hub *Hub, db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        user := auth.GetUser(r.Context())
        if user == nil {
            http.Error(w, "Unauthorized", 401)
            return
        }
        
        conn, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
            return
        }
        
        client := &Client{
            hub:    hub,
            userID: user.ID,
            conn:   conn,
            send:   make(chan []byte, 256),
        }
        
        hub.Register(client)
        
        go client.WritePump()
        go client.ReadPump()
    }
}
```

### 4.4 Integrate WebSocket with Message Sending

**Modify POST /conversations/:userID/messages**

After successfully creating the message:
1. Build OutgoingMessage struct
2. Call `hub.SendToUser(recipientID, msg)`
3. Also call `hub.SendToUser(senderID, msg)` — so sender's other devices see it too

This ensures multi-device sync for both parties.

### 4.5 Update Chat UI for WebSocket

Location: `web/templates/chat.html` modifications

On page load (after auth confirmed):
1. Open WebSocket connection to `/ws`
2. On message event, parse JSON
3. If message type is "message":
   - If `from` matches currently open conversation, append to message list
   - Update conversation list sidebar (move this conversation to top, update preview)
   - If conversation not in sidebar yet, add it

Remove polling code.

**Handle reconnection**
- On WebSocket close, attempt reconnect after 2 seconds
- Exponential backoff up to 30 seconds
- On reconnect, refresh current conversation to catch any missed messages

### 4.6 Phase 4 Testing Checklist

- [ ] WebSocket connects successfully after login
- [ ] WebSocket rejects unauthenticated requests
- [ ] User A sends message, User B receives instantly (no refresh)
- [ ] User A has two tabs open, sends from one, message appears in both
- [ ] User B has two devices, message appears on both
- [ ] Closing and reopening tab reconnects WebSocket
- [ ] Conversation list updates in real-time when new message arrives
- [ ] Messages from user not in sidebar adds them to sidebar
- [ ] No polling requests visible in network tab
- [ ] Server handles client disconnect gracefully (no goroutine leaks)

---

## Phase 5: Cleanup and Hardening

### 5.1 Message Expiry

**Background goroutine in main.go**

On startup, launch a goroutine that:
1. Runs immediately, then every hour (use `time.Ticker`)
2. Executes: `DELETE FROM messages WHERE created_at < datetime('now', '-30 days')`
3. Log number of deleted rows

Note: If a conversation has all messages expired, it will naturally disappear from the conversation list since the query only returns conversations with messages.

### 5.2 Session Expiry

**Background goroutine**

Similar pattern, run every hour:

```sql
DELETE FROM sessions WHERE expires_at < datetime('now')
```

This prevents unbounded session table growth.

### 5.3 CSRF Protection

For form submissions, implement CSRF tokens:

**Option A: Synchroniser token**
- Generate random token per session, store in sessions table
- Include as hidden field in all forms
- Validate on POST requests

**Option B: Double-submit cookie**
- Set a random CSRF cookie
- Include same value in form or header
- Compare on server

**Option C: SameSite cookies (already in place)**
- If all users are on modern browsers and you're HTTPS-only, SameSite=Lax on session cookie provides good protection
- This is the simplest approach for a family app

**Recommendation:** Rely on SameSite=Lax for forms, but for any JSON API endpoints called via JavaScript, require the session cookie (which won't be sent cross-origin due to SameSite).

### 5.4 Input Validation and Sanitisation

**Server-side**
- Username: alphanumeric, 3-30 chars
- Display name: 1-50 chars, trim whitespace
- Password: minimum 8 chars (or whatever policy you want)
- Message content: 1-4000 chars, trim whitespace

**HTML output**
- Go's `html/template` auto-escapes by default—ensure you're using it correctly
- Never use `template.HTML` on user content

**Database**
- Using parameterised queries (which you should be) prevents SQL injection

### 5.5 Rate Limiting (Optional)

For a family app, probably unnecessary, but if desired:

**Simple in-memory approach**
- Map of userID → slice of timestamps
- On each message send, check if more than N messages in last M seconds
- If exceeded, return 429 Too Many Requests
- Clean old timestamps periodically

**Middleware placement**
- Apply to POST endpoints only
- Or specifically to message sending

### 5.6 HTTPS and Secure Cookies

**Production deployment**
- Run behind Caddy or nginx with automatic HTTPS
- Or use `autocert` package for built-in HTTPS

**Cookie adjustments for production**
- Set `Secure: true` on session cookie
- Ensure `SameSite: Lax` is set

**Configuration**
- Add `BaseURL` to config (for building absolute URLs if needed)
- Add `SecureCookies` boolean flag

### 5.7 Mobile-Friendly CSS

**Responsive layout**
- Sidebar hidden by default on mobile, toggle with hamburger icon
- Or sidebar becomes bottom tabs
- Or: single-column layout, conversation list → tap → message view → back button

**Touch considerations**
- Adequate tap target sizes (44px minimum)
- No hover-dependent interactions

**Viewport**
- Proper viewport meta tag
- Handle virtual keyboard appearance (input at bottom should remain visible)

**Tailwind approach**
- Use responsive prefixes: `md:`, `lg:`
- Mobile-first, add complexity for larger screens

### 5.8 Additional Hardening

- Passwords never logged
- Session tokens never logged
- Database file not in web-accessible directory
- No directory listing on static files
- Errors don't leak stack traces to client (log internally, return generic message)
- WebSocket connection limits (optional, prevent resource exhaustion)
- Request body size limits (default `http.Server` has 1MB for forms, but set explicitly)

### 5.9 Phase 5 Testing Checklist

- [ ] Messages older than 30 days are deleted (backdate some manually to test)
- [ ] Expired sessions are cleaned up
- [ ] CSRF tokens work (if implemented)
- [ ] Invalid input rejected with clear error messages
- [ ] XSS attempt in message content is escaped properly
- [ ] App works over HTTPS
- [ ] Session cookie has Secure and HttpOnly flags
- [ ] App is usable on mobile phone
- [ ] Conversation sidebar works on mobile
- [ ] Sending message works on mobile
- [ ] No console errors in browser

---

## Deployment Notes

### Build

```bash
CGO_ENABLED=0 go build -o wantok ./cmd/server
```

(If using pure-Go SQLite driver; otherwise `CGO_ENABLED=1`)

### Systemd Service

Create `/etc/systemd/system/wantok.service`:

```ini
[Unit]
Description=Wantok Family Chat
After=network.target

[Service]
Type=simple
User=wantok
Group=wantok
WorkingDirectory=/opt/wantok
ExecStart=/opt/wantok/wantok
Restart=on-failure
RestartSec=5
Environment=WANTOK_DATABASE_PATH=/opt/wantok/data/wantok.db
Environment=WANTOK_LISTEN_ADDR=127.0.0.1:8080
Environment=WANTOK_SECURE_COOKIES=true

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable wantok
sudo systemctl start wantok
```

### Caddy Configuration

```
chat.yourdomain.com {
    reverse_proxy localhost:8080
}
```

Caddy automatically handles HTTPS certificates and WebSocket proxying.

### Backup

SQLite database file can be backed up with `.backup` command or just copy while using WAL mode.

Simple daily backup script:

```bash
#!/bin/bash
sqlite3 /opt/wantok/data/wantok.db ".backup /opt/wantok/backups/wantok-$(date +%Y%m%d).db"
find /opt/wantok/backups -name "*.db" -mtime +7 -delete
```

### Monitoring

Simple health check endpoint: `GET /health` returns 200.

Log to stdout, collect with journald:

```bash
journalctl -u wantok -f
```