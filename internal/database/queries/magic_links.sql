-- name: CreateMagicLink :one
INSERT INTO magic_links (token, user_id, expires_at)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetMagicLinkWithUser :one
SELECT
    m.token,
    m.user_id,
    m.created_at,
    m.expires_at,
    u.id,
    u.username,
    u.display_name,
    u.email,
    u.is_admin
FROM magic_links m
JOIN users u ON m.user_id = u.id
WHERE m.token = ?
  AND m.expires_at > datetime('now');

-- name: DeleteMagicLink :exec
DELETE FROM magic_links WHERE token = ?;

-- name: DeleteUserMagicLinks :exec
DELETE FROM magic_links WHERE user_id = ?;

-- name: DeleteExpiredMagicLinks :execresult
DELETE FROM magic_links WHERE expires_at < datetime('now');

-- name: CountRecentMagicLinksByUserID :one
SELECT COUNT(*) FROM magic_links
WHERE user_id = ?
  AND created_at > datetime('now', '-1 hour');
