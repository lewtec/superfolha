-- name: CreateUser :one
INSERT INTO users (email, password_hash)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: CreateProject :one
INSERT INTO projects (id, user_id, name, git_path)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetProject :one
SELECT * FROM projects
WHERE id = $1;

-- name: GetUserProjects :many
SELECT * FROM projects
WHERE user_id = $1
ORDER BY updated_at DESC;

-- name: UpdateProjectTimestamp :exec
UPDATE projects
SET updated_at = NOW()
WHERE id = $1;

-- name: DeleteProject :exec
DELETE FROM projects
WHERE id = $1;
