-- +goose Up
-- Add email column to users (nullable for existing users)
ALTER TABLE users ADD COLUMN email TEXT UNIQUE;

-- Invitations table for email-based registration
CREATE TABLE invitations (
    token TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    invited_by INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at TEXT NOT NULL
);

CREATE INDEX idx_invitations_email ON invitations(email);
CREATE INDEX idx_invitations_expires_at ON invitations(expires_at);

-- Magic links table for passwordless login
CREATE TABLE magic_links (
    token TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at TEXT NOT NULL
);

CREATE INDEX idx_magic_links_user_id ON magic_links(user_id);
CREATE INDEX idx_magic_links_expires_at ON magic_links(expires_at);

-- +goose Down
DROP INDEX idx_magic_links_expires_at;
DROP INDEX idx_magic_links_user_id;
DROP TABLE magic_links;

DROP INDEX idx_invitations_expires_at;
DROP INDEX idx_invitations_email;
DROP TABLE invitations;

-- SQLite doesn't support DROP COLUMN, so we need to recreate the table
CREATE TABLE users_backup AS SELECT id, username, display_name, password_hash, is_admin, created_at FROM users;
DROP TABLE users;
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    is_admin INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
INSERT INTO users SELECT * FROM users_backup;
DROP TABLE users_backup;
