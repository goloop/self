-- name: CreateUser :one
INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING *;
-- name: UserByEmail :one
SELECT * FROM users WHERE email = $1;
-- name: UserByID :one
SELECT * FROM users WHERE id = $1;
-- name: CreateNote :one
INSERT INTO notes (user_id, title) VALUES ($1, $2) RETURNING *;
-- name: NotesByUser :many
SELECT * FROM notes WHERE user_id = $1 ORDER BY id DESC;
