package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"goloop.one/handbook/010-whole-stack/internal/store"

	"github.com/goloop/auth"

	_ "github.com/lib/pq"
)

const testSecret = "a-32-byte-or-longer-signing-secret!!"

// TestFlow signs up, logs in and creates a protected note against a real DB,
// skipping when PGC_DATABASE_URL is unset (run `pgc migrate` first).
func TestFlow(t *testing.T) {
	url := os.Getenv("PGC_DATABASE_URL")
	if url == "" {
		t.Skip("set PGC_DATABASE_URL and run `pgc migrate` to test against a database")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, _ = db.Exec("TRUNCATE users, notes RESTART IDENTITY CASCADE")

	a := &api{
		cfg:    Config{AuthSecret: testSecret},
		hasher: auth.NewPBKDF2(),
		tm:     auth.NewTokenManager([]byte(testSecret)),
		db:     db,
		q:      store.New(db),
	}
	a.routes()

	// signup returns a token
	rec := httptest.NewRecorder()
	a.router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/signup",
		strings.NewReader(`{"Email":"a@b.co","Password":"password1"}`)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("signup = %d: %s", rec.Code, rec.Body)
	}
	var body struct{ Token string }
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Token == "" {
		t.Fatal("no token returned")
	}

	// a protected create note succeeds with the token
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/notes", strings.NewReader(`{"Title":"first"}`))
	req.Header.Set("Authorization", "Bearer "+body.Token)
	a.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create note = %d: %s", rec.Code, rec.Body)
	}

	// the same route without a token is rejected
	rec = httptest.NewRecorder()
	a.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/notes", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated list = %d, want 401", rec.Code)
	}
}
