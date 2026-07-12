package main

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"goloop.one/handbook/004-postgresql/internal/store"

	_ "github.com/lib/pq"
)

// TestNotesRoundTrip runs against a real database when PGC_DATABASE_URL is set,
// and skips otherwise, so `go test` passes with or without a database. The
// migrations must already be applied (pgc migrate).
func TestNotesRoundTrip(t *testing.T) {
	url := os.Getenv("PGC_DATABASE_URL")
	if url == "" {
		t.Skip("set PGC_DATABASE_URL and run `pgc migrate` to test against a database")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	q := store.New(db)
	n, err := q.CreateNote(ctx, "Test note", "body", []string{"t1", "t2"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := q.NoteByID(ctx, n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Test note" || len(got.Tags) != 2 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}
