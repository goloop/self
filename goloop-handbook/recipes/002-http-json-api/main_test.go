package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestReadRoutes covers example A: JSON, a path parameter and a clean 404,
// plus the security and custom headers every response carries.
func TestReadRoutes(t *testing.T) {
	h := newRouter(newStore())
	cases := []struct {
		method, path, wantBodyContains string
		wantStatus                     int
	}{
		{"GET", "/health", `"status":"ok"`, 200},
		{"GET", "/users/1", `"email":"ada@example.com"`, 200},
		{"GET", "/users/999", "user not found", 404},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, nil))
		if rec.Code != c.wantStatus {
			t.Errorf("%s %s: status %d, want %d", c.method, c.path, rec.Code, c.wantStatus)
		}
		if !strings.Contains(rec.Body.String(), c.wantBodyContains) {
			t.Errorf("%s %s: body %q missing %q", c.method, c.path, rec.Body.String(), c.wantBodyContains)
		}
		if rec.Header().Get("X-Content-Type-Options") == "" {
			t.Errorf("%s %s: security header missing", c.method, c.path)
		}
		if rec.Header().Get("X-Api-Version") != "v1" {
			t.Errorf("%s %s: custom middleware header missing", c.method, c.path)
		}
	}
}

// TestWriteStatuses covers example B: 201 Created with a Location header on
// POST, and 204 No Content on DELETE.
func TestWriteStatuses(t *testing.T) {
	h := newRouter(newStore())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/users",
		strings.NewReader(`{"name":"Grace","email":"grace@example.com"}`)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want 201", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/users/2" {
		t.Fatalf("Location = %q, want /users/2", loc)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/users/1", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want 204", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("204 body not empty: %q", rec.Body.String())
	}
}

// TestListWithQueryParams covers example D: qp reads and bounds limit/offset and
// filters by q.
func TestListWithQueryParams(t *testing.T) {
	s := newStore()
	s.add(user{Name: "Grace", Email: "grace@example.com"})
	s.add(user{Name: "Alan", Email: "alan@example.com"})
	h := newRouter(s)

	// A valid in-range limit is honored: limit=2 returns two rows.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/users?limit=2", nil))
	var out struct {
		Users []user `json:"users"`
		Limit int    `json:"limit"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Limit != 2 || len(out.Users) != 2 {
		t.Errorf("limit=2 -> limit %d, users %d; want 2 and 2", out.Limit, len(out.Users))
	}

	// An out-of-range limit is rejected and falls back to the default (20), not
	// clamped: qp.Between drops the bad value rather than truncating it.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/users?limit=999", nil))
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Limit != 20 {
		t.Errorf("limit=999 -> %d, want default 20", out.Limit)
	}
	if len(out.Users) != 3 {
		t.Errorf("users = %d, want 3", len(out.Users))
	}

	// q filters by name substring, case-insensitively.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/users?q=gra", nil))
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Users) != 1 || out.Users[0].Name != "Grace" {
		t.Errorf("q filter = %+v", out.Users)
	}
}
