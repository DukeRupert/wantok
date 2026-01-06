# Database Schema

SQLite database with WAL mode enabled for better concurrent read performance.

## Tables

### users

Stores family member accounts.

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

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | INTEGER | PRIMARY KEY, AUTO | Unique identifier |
| username | TEXT | UNIQUE, NOT NULL | Login credential, 3-30 alphanumeric |
| display_name | TEXT | NOT NULL | Shown in UI, 1-50 chars |
| password_hash | TEXT | NOT NULL | bcrypt hash, cost 12 |
| is_admin | INTEGER | NOT NULL, DEFAULT 0 | 1 = admin, 0 = regular user |
| created_at | TEXT | NOT NULL | ISO 8601 timestamp |

---

### sessions

Stores authentication tokens.

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

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| token | TEXT | PRIMARY KEY | 64-char hex string (32 random bytes) |
| user_id | INTEGER | FK → users.id, CASCADE | Owning user |
| created_at | TEXT | NOT NULL | When session was created |
| expires_at | TEXT | NOT NULL | When session becomes invalid |

**Indexes:**
- `idx_sessions_user_id` — For deleting all sessions when user is deleted
- `idx_sessions_expires_at` — For cleanup job

**Notes:**
- Sessions expire 30 days after creation
- Deleting a user cascades to delete their sessions
- Background job cleans expired sessions hourly

---

### messages

Stores conversation messages.

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

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | INTEGER | PRIMARY KEY, AUTO | Unique identifier |
| sender_id | INTEGER | FK → users.id, CASCADE | Who sent it |
| recipient_id | INTEGER | FK → users.id, CASCADE | Who receives it |
| content | TEXT | NOT NULL | Message text, 1-4000 chars |
| created_at | TEXT | NOT NULL | ISO 8601 timestamp |

**Indexes:**
- `idx_messages_sender_id` — For finding sent messages
- `idx_messages_recipient_id` — For finding received messages
- `idx_messages_created_at` — For 30-day cleanup and pagination

**Notes:**
- Messages older than 30 days are deleted by background job
- Deleting a user cascades to delete their messages
- No separate "conversations" table; conversations are derived from message pairs

---

## Migrations

Use a simple versioned migration system.

### schema_migrations table

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY
);
```

### Migration files

Track migrations as numbered SQL or Go functions:

| Version | Description |
|---------|-------------|
| 001 | Create users table |
| 002 | Create sessions table with indexes |
| 003 | Create messages table with indexes |

### Migration runner pseudocode

```go
func RunMigrations(db *sql.DB) error {
    // Create schema_migrations if not exists
    // Get current version (0 if table empty)
    // For each migration > current version:
    //   - Execute migration
    //   - Update version
    return nil
}
```

---

## Common Queries

### Validate session

```sql
SELECT u.id, u.username, u.display_name, u.is_admin
FROM sessions s
JOIN users u ON s.user_id = u.id
WHERE s.token = ?
  AND s.expires_at > datetime('now');
```

### Get conversation list

```sql
WITH ranked_messages AS (
    SELECT 
        CASE 
            WHEN sender_id = ? THEN recipient_id 
            ELSE sender_id 
        END AS other_user_id,
        content,
        created_at,
        sender_id,
        ROW_NUMBER() OVER (
            PARTITION BY CASE 
                WHEN sender_id = ? THEN recipient_id 
                ELSE sender_id 
            END 
            ORDER BY created_at DESC
        ) AS rn
    FROM messages
    WHERE sender_id = ? OR recipient_id = ?
)
SELECT 
    rm.other_user_id,
    u.display_name,
    rm.content AS last_message,
    rm.created_at AS last_message_at,
    rm.sender_id = ? AS is_from_me
FROM ranked_messages rm
JOIN users u ON rm.other_user_id = u.id
WHERE rm.rn = 1
ORDER BY rm.created_at DESC;
```

### Get messages in conversation

```sql
SELECT 
    m.id,
    m.sender_id,
    u.display_name AS sender_display_name,
    m.recipient_id,
    m.content,
    m.created_at
FROM messages m
JOIN users u ON m.sender_id = u.id
WHERE (m.sender_id = ? AND m.recipient_id = ?)
   OR (m.sender_id = ? AND m.recipient_id = ?)
ORDER BY m.created_at DESC
LIMIT ?
OFFSET ?;
```

### Cleanup old messages

```sql
DELETE FROM messages 
WHERE created_at < datetime('now', '-30 days');
```

### Cleanup expired sessions

```sql
DELETE FROM sessions 
WHERE expires_at < datetime('now');
```

---

## SQLite Configuration

Enable on connection:

```sql
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
```

**journal_mode = WAL** — Write-Ahead Logging for better concurrent reads

**foreign_keys = ON** — Enforce referential integrity (SQLite doesn't by default)

**busy_timeout = 5000** — Wait up to 5 seconds if database is locked