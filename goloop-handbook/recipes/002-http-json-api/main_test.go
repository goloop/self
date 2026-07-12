package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRoutes exercises the router directly with httptest - no network needed.
func TestRoutes(t *testing.T) {
	h := newRouter()

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
		// The security-headers middleware runs on every response.
		if rec.Header().Get("X-Content-Type-Options") == "" {
			t.Errorf("%s %s: security header missing", c.method, c.path)
		}
	}
}

func TestRequestIDHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	newRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Header().Get("X-Request-Id") == "" {
		t.Error("request id header not set")
	}
}
