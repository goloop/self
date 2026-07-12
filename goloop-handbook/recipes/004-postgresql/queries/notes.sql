-- name: CreateNote :one
-- Inserts a note and returns the whole row, with the generated id and timestamp.
INSERT INTO notes (title, body, tags) VALUES ($1, $2, $3) RETURNING *;

-- name: NoteByID :one
-- Returns one note by primary key.
SELECT * FROM notes WHERE id = $1;

-- name: ListNotes :many
-- Lists notes, newest first, with a limit.
SELECT * FROM notes ORDER BY id DESC LIMIT $1;

-- name: CountNotes :one
-- Total number of notes.
SELECT count(*) FROM notes;

-- name: SearchNotes :many
-- Case-insensitive title search.
SELECT * FROM notes WHERE title ILIKE '%' || $1 || '%' ORDER BY id DESC;
