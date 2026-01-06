-- name: CreateSession :one
INSERT INTO sessions (token, user_id, expires_at)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetSessionWithUser :one
SELECT
    s.token,
    s.user_id,
    s.created_at,
    s.expires_at,
    u.id AS user_id,
    u.username,
    u.display_name,
    u.is_admin
FROM sessions s
JOIN users u ON s.user_id = u.id
WHERE s.token = ?
  AND s.expires_at > datetime('now');

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = ?;

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = ?;

-- name: DeleteExpiredSessions :execresult
DELETE FROM sessions WHERE expires_at < datetime('now');