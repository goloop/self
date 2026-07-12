[« Validate and clean](03-validate-and-clean.md) · [Contents](../main.md) · [Language models »](05-ai.md)

---

# 04. Typed PostgreSQL with migrations

**Task.** Evolve a database schema over time, and query it from Go without
hand-writing scan code or risking a typo in a column name. When you rename a
column, the code that used it should stop compiling, not fail at runtime.

**Module.** [`pgc`](https://github.com/goloop/pgc) is a SQL-to-Go compiler for
PostgreSQL with its own migrations. It has two jobs, run from the command line:

- `pgc migrate` applies the `.sql` files in `migrations/` in order;
- `pgc generate` turns the queries in `queries/` into typed Go methods.

**Recipe.** [`recipes/004-postgresql`](../recipes/004-postgresql/)

## The idea

You write two kinds of SQL by hand: migrations that shape the schema, and
queries that read and write it. `pgc` reads both. From a migration it knows the
table; from a query it generates a Go method with typed parameters and a typed
result. The generated package depends only on `database/sql`, so the program
below imports no GoLoop package at all: it just uses the code `pgc` wrote.

The schema is one migration file, `migrations/001_notes.sql`:

```sql
CREATE TABLE notes (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title      text NOT NULL,
    body       text NOT NULL DEFAULT '',
    tags       text[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);
```

The queries are annotated SQL, `queries/notes.sql`. The `:one` / `:many` tag
tells `pgc` whether the method returns a single row or a slice:

```sql
-- name: CreateNote :one
INSERT INTO notes (title, body, tags) VALUES ($1, $2, $3) RETURNING *;

-- name: SearchNotes :many
SELECT * FROM notes WHERE title ILIKE '%' || $1 || '%' ORDER BY id DESC;
```

`pgc generate` turns that into a typed `Note` struct and typed methods:

```go
type Note struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Tags      []string  `json:"tags"` // Postgres text[] maps to []string
	CreatedAt time.Time `json:"created_at"`
}

func (q *Queries) CreateNote(ctx context.Context, title, body string, tags []string) (Note, error)
func (q *Queries) SearchNotes(ctx context.Context, arg1 string) ([]Note, error)
```

## Example A - write

`CreateNote` runs `INSERT ... RETURNING *` and hands back a fully typed `Note`,
including the database-generated `id` and `created_at`:

```go
q := store.New(db) // wraps *sql.DB, safe to share
n, err := q.CreateNote(ctx, "Reading list", "Books to read this month.",
	[]string{"personal", "books"})
```

## Example B - read

Single-row and multi-row reads are ordinary method calls that return typed rows:

```go
got, _ := q.NoteByID(ctx, n.ID)   // Note
list, _ := q.ListNotes(ctx, 10)   // []Note
total, _ := q.CountNotes(ctx)     // *int64
```

## Example C - search, with an array

`SearchNotes` takes a parameter, and the `text[]` column comes back as a Go
`[]string` with no scanning code on your side:

```go
found, _ := q.SearchNotes(ctx, "list") // []Note, each with .Tags []string
```

## Example D - a transaction

Some writes must happen together or not at all. `pgc` generates a `WithTx`
method: `q.WithTx(tx)` returns a `Queries` bound to a `*sql.Tx`, so the same
typed methods now run inside the transaction. Both `*sql.DB` and `*sql.Tx`
satisfy the small `DBTX` interface the generated code needs, so there is no
second set of methods to learn:

```go
tx, _ := db.BeginTx(ctx, nil)
qtx := q.WithTx(tx) // the same methods, now inside tx

if _, err := qtx.CreateNote(ctx, "Groceries", "Milk, bread.", []string{"home"}); err != nil {
	_ = tx.Rollback() // one failure undoes the whole batch
	return err
}
if _, err := qtx.CreateNote(ctx, "Standup", "Blockers and plan.", []string{"work"}); err != nil {
	_ = tx.Rollback()
	return err
}
_ = tx.Commit() // both notes land together
```

Roll back instead of committing and the writes vanish: a `CreateNote` inside a
rolled-back `tx` is never visible, so `CountNotes` is unchanged afterwards.

## Execution report

Migrations applied, tested against a real PostgreSQL, then run:

```text
$ pgc migrate
applied 001_notes.sql

$ go test ./...            # against the DB; skips gracefully when it is unset
ok  	goloop.one/handbook/004-postgresql	0.010s

$ go run .
A. write (CreateNote -> typed Note):
   inserted id=1 title="Reading list" tags=[personal books]
B. read (NoteByID, ListNotes, CountNotes):
   NoteByID(1) -> "Reading list"
   - #2 "Release checklist" [work]
   - #1 "Reading list" [personal books]
   total = 2
C. search (SearchNotes, parameter + text[] -> []string):
   match #2 "Release checklist" tags=[work]
   match #1 "Reading list" tags=[personal books]
D. transaction (WithTx, commit then rollback):
   committed 2 notes: 2 -> 4
   after rollback: 4 -> 4 (unchanged)
```

The insert returned a typed `Note` with the generated id; the reads returned
`Note` and `[]Note`; the search matched both titles that contain "list"; and
`tags` moved between Postgres and Go as `[]string` throughout. The committed
transaction added two notes at once (2 -> 4); the rolled-back one left the
count untouched (4 -> 4), proving the writes were atomic.

## What you learned

- `pgc` has two command-line jobs: `pgc migrate` (apply the schema) and
  `pgc generate` (compile the queries into typed Go).
- Annotate a query with `-- name: X :one` or `:many`; `pgc` writes a typed
  method and struct. `INSERT ... RETURNING *` becomes a typed write.
- The generated package uses only `database/sql`; a Postgres `text[]` maps to a
  Go `[]string`.
- `q.WithTx(tx)` runs the same typed methods inside a `*sql.Tx`: commit and the
  batch lands together, roll back and it never happened.
- Because the columns are typed in Go, renaming one in a migration breaks the
  build, not production.

Part II continues with asking a language model about this data.

---

[« Validate and clean](03-validate-and-clean.md) · [Contents](../main.md) · [Language models »](05-ai.md)
