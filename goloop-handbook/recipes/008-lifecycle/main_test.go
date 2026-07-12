package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goloop/observe"
)

// TestProbes checks the liveness and readiness handlers directly.
func TestProbes(t *testing.T) {
	reg := observe.New(observe.WithService("t"))
	reg.Check("ok", func(context.Context) error { return nil })

	for _, path := range []string{"/healthz", "/readyz"} {
		rec := httptest.NewRecorder()
		var h http.Handler
		if path == "/healthz" {
			h = reg.HealthHandler()
		} else {
			h = reg.ReadyHandler()
		}
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s = %d, want 200", path, rec.Code)
		}
	}
}

// TestReadinessGate covers example D: a check tied to a dependency makes
// /readyz report 503 while the dependency is down and 200 once it is up.
func TestReadinessGate(t *testing.T) {
	var dbUp atomic.Bool
	reg := observe.New(observe.WithService("t"))
	reg.Check("database", func(context.Context) error {
		if !dbUp.Load() {
			return errDependencyDown
		}
		return nil
	})
	h := reg.ReadyHandler()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("db down: /readyz = %d, want 503", rec.Code)
	}

	dbUp.Store(true)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("db up: /readyz = %d, want 200", rec.Code)
	}
}

// TestLifecycleShutsDown starts the whole service and confirms it returns
// cleanly when its context is cancelled.
func TestLifecycleShutsDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if err := run(ctx); err != nil {
		t.Fatalf("run returned %v, want clean shutdown", err)
	}
}
