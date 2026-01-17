-- name: CreateInvitation :one
INSERT INTO invitations (token, email, invited_by, expires_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetInvitationByToken :one
SELECT * FROM invitations
WHERE token = ?
  AND expires_at > datetime('now');

-- name: GetInvitationByEmail :one
SELECT * FROM invitations
WHERE email = ?
  AND expires_at > datetime('now');

-- name: DeleteInvitation :exec
DELETE FROM invitations WHERE token = ?;

-- name: DeleteExpiredInvitations :execresult
DELETE FROM invitations WHERE expires_at < datetime('now');

-- name: CountRecentInvitationsByEmail :one
SELECT COUNT(*) FROM invitations
WHERE email = ?
  AND created_at > datetime('now', '-1 hour');
