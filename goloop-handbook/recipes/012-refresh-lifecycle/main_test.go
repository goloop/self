package main

import (
	"testing"
	"time"

	"github.com/goloop/auth"
	"github.com/goloop/auth/authtest"
)

// The recipe uses auth.NewMemoryRefreshStore. This is the test the chapter
// leans on: authtest runs the whole refresh-store contract - atomic rotation
// under concurrency, reuse detection, grace, expiry, the subject index - so a
// store is proven, not hoped for. Point it at your Redis or Postgres store and
// the same suite holds it to the same contract.
func TestMemoryStoreConformance(t *testing.T) {
	authtest.RefreshStore(t, func(t *testing.T) auth.RefreshStore {
		return auth.NewMemoryRefreshStore(auth.WithGrace(0))
	})
}

// The same store with a grace window open, so the grace branch is exercised
// too. A real store must pass both shapes.
func TestMemoryStoreConformanceWithGrace(t *testing.T) {
	authtest.RefreshStore(t, func(t *testing.T) auth.RefreshStore {
		return auth.NewMemoryRefreshStore(auth.WithGrace(time.Minute))
	})
}

// The recipe's own narrative runs clean end to end.
func TestRunPlaysThrough(t *testing.T) {
	if err := run(); err != nil {
		t.Fatal(err)
	}
}
