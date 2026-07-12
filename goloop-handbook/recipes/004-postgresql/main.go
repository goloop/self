// Recipe 004: typed PostgreSQL with migrations.
//
// The task: evolve a schema with migrations and query it from Go without
// hand-writing scan code or risking a typo in a column name. pgc has two jobs:
//
//	pgc migrate   - apply the SQL files in migrations/ in order;
//	pgc generate  - turn the SQL in queries/ into typed Go methods.
//
// The generated package depends only on database/sql, so this program imports
// no goloop package at all - it just uses the code pgc wrote.
//
// Four examples, all against a real database:
//
//	A. write  - CreateNote inserts and returns a typed Note (INSERT RETURNING);
//	B. read   - NoteByID, ListNotes and CountNotes return typed rows;
//	C. search - SearchNotes takes a parameter, and tags round-trips as []string;
//	D. tx     - q.WithTx runs several writes atomically; a rollback undoes them.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"goloop.one/handbook/004-postgresql/internal/store"

	_ "github.com/lib/pq"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "recipe:", err)
		os.Exit(1)
	}
}

func run() error {
	db, err := sql.Open("postgres", os.Getenv("PGC_DATABASE_URL"))
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	// A single Queries value, safe to share, wraps *sql.DB.
	q := store.New(db)

	// Example A: write. CreateNote runs INSERT ... RETURNING * and hands back a
	// fully typed Note, including the database-generated id and created_at.
	fmt.Println("A. write (CreateNote -> typed Note):")
	n1, err := q.CreateNote(ctx, "Reading list", "Books to read this month.", []string{"personal", "books"})
	if err != nil {
		return err
	}
	_, _ = q.CreateNote(ctx, "Release checklist", "Tag, push, announce.", []string{"work"})
	fmt.Printf("   inserted id=%d title=%q tags=%v\n", n1.ID, n1.Title, n1.Tags)

	// Example B: read. Typed single-row and multi-row queries.
	fmt.Println("B. read (NoteByID, ListNotes, CountNotes):")
	got, err := q.NoteByID(ctx, n1.ID)
	if err != nil {
		return err
	}
	fmt.Printf("   NoteByID(%d) -> %q\n", n1.ID, got.Title)
	list, err := q.ListNotes(ctx, 10)
	if err != nil {
		return err
	}
	for _, n := range list {
		fmt.Printf("   - #%d %q %v\n", n.ID, n.Title, n.Tags)
	}
	if total, err := q.CountNotes(ctx); err == nil && total != nil {
		fmt.Printf("   total = %d\n", *total)
	}

	// Example C: search with a parameter; tags come back as a Go []string.
	fmt.Println("C. search (SearchNotes, parameter + text[] -> []string):")
	found, err := q.SearchNotes(ctx, "list")
	if err != nil {
		return err
	}
	for _, n := range found {
		fmt.Printf("   match #%d %q tags=%v\n", n.ID, n.Title, n.Tags)
	}

	// Example D: a transaction. q.WithTx(tx) returns a Queries bound to the
	// transaction, so several writes either all land or none do. The first tx
	// commits two notes; the second inserts and then rolls back, and CountNotes
	// proves the rolled-back rows never existed.
	fmt.Println("D. transaction (WithTx, commit then rollback):")
	before := count(ctx, q)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	qtx := q.WithTx(tx)
	if _, err := qtx.CreateNote(ctx, "Groceries", "Milk, bread.", []string{"home"}); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := qtx.CreateNote(ctx, "Standup", "Blockers and plan.", []string{"work"}); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("   committed 2 notes: %d -> %d\n", before, count(ctx, q))

	// Now a transaction we deliberately roll back: the insert is invisible after.
	rolled := count(ctx, q)
	tx2, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := q.WithTx(tx2).CreateNote(ctx, "Never saved", "Rolled back.", []string{"temp"}); err != nil {
		_ = tx2.Rollback()
		return err
	}
	_ = tx2.Rollback()
	fmt.Printf("   after rollback: %d -> %d (unchanged)\n", rolled, count(ctx, q))
	return nil
}

// count returns the number of notes, or -1 on error, for the report.
func count(ctx context.Context, q *store.Queries) int64 {
	total, err := q.CountNotes(ctx)
	if err != nil || total == nil {
		return -1
	}
	return *total
}
