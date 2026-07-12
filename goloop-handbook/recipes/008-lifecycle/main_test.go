package main

import (
	"context"
	"net/http"
	"net/http/httptest"
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

// TestLifecycleShutsDown starts the whole service and confirms it returns
// cleanly when its context is cancelled.
func TestLifecycleShutsDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if err := run(ctx); err != nil {
		t.Fatalf("run returned %v, want clean shutdown", err)
	}
}
