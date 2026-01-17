-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ?;

-- name: ListUsers :many
SELECT * FROM users ORDER BY display_name;

-- name: ListUsersExcept :many
SELECT * FROM users WHERE id != ? ORDER BY display_name;

-- name: CreateUser :one
INSERT INTO users (username, display_name, password_hash, is_admin)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: UpdateUser :exec
UPDATE users
SET display_name = ?, password_hash = ?, is_admin = ?
WHERE id = ?;

-- name: UpdateUserDisplayName :exec
UPDATE users
SET display_name = ?
WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: CreateUserWithEmail :one
INSERT INTO users (username, display_name, password_hash, email, is_admin)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateUserEmail :exec
UPDATE users SET email = ? WHERE id = ?;