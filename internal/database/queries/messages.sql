-- name: CreateMessage :one
INSERT INTO messages (sender_id, recipient_id, content)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetConversationMessages :many
SELECT
    m.id,
    m.sender_id,
    m.recipient_id,
    m.content,
    m.created_at,
    u.display_name AS sender_display_name
FROM messages m
JOIN users u ON m.sender_id = u.id
WHERE (m.sender_id = ? AND m.recipient_id = ?)
   OR (m.sender_id = ? AND m.recipient_id = ?)
ORDER BY m.created_at DESC
LIMIT ?
OFFSET ?;

-- name: GetRecentMessagePerUser :many
SELECT
    m.*,
    u.display_name AS other_user_display_name
FROM messages m
JOIN users u ON u.id = CASE
    WHEN m.sender_id = ? THEN m.recipient_id
    ELSE m.sender_id
END
WHERE m.sender_id = ? OR m.recipient_id = ?
ORDER BY m.created_at DESC;

-- name: DeleteOldMessages :execresult
DELETE FROM messages
WHERE created_at < datetime('now', '-30 days');