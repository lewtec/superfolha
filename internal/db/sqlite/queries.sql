-- name: CreateUser :one
INSERT INTO users (id, email, password_hash)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE lower(email) = lower(?);

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = ?;

-- name: CreateProject :one
INSERT INTO projects (id, user_id, name, git_path)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetProject :one
SELECT * FROM projects
WHERE id = ?;

-- name: GetUserProjects :many
SELECT * FROM projects
WHERE user_id = ?
ORDER BY updated_at DESC;

-- name: UpdateProjectTimestamp :exec
UPDATE projects
SET updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteProject :exec
DELETE FROM projects
WHERE id = ?;
