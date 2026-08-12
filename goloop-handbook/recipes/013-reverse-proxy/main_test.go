package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/goloop/middlewares"
)

// The heart of the recipe: behind a proxy without a trusted-proxy policy, two
// clients collapse to one rate-limit key; with the policy, they separate. This
// is the bug a real project shipped, so it is pinned.
func TestTrustedProxyDecidesTheKey(t *testing.T) {
	naive := router(false)
	if a, b := behindProxy(naive, "203.0.113.1"), behindProxy(naive, "203.0.113.2"); a["rate_limit_key"] != b["rate_limit_key"] {
		t.Errorf("without a trusted proxy the keys already differ (%s vs %s) - "+
			"the collapse the chapter warns about is not reproduced",
			a["rate_limit_key"], b["rate_limit_key"])
	}

	fixed := router(true)
	a := behindProxy(fixed, "203.0.113.1")
	b := behindProxy(fixed, "203.0.113.2")
	if a["rate_limit_key"] == b["rate_limit_key"] {
		t.Errorf("with a trusted proxy the two clients still share a key %s - "+
			"the fix did not take", a["rate_limit_key"])
	}
	if a["rate_limit_key"] != "203.0.113.1" {
		t.Errorf("resolved key = %s, want the forwarded client address",
			a["rate_limit_key"])
	}
}

// A forged X-Forwarded-For from a client with no trusted proxy in front is not
// believed: the safe default is what makes the resolved address trustworthy.
func TestForgedHeaderIsNotBelieved(t *testing.T) {
	// The client connects directly (its own address is not in the trusted
	// range) and lies in the header.
	got := behindProxyFrom(router(true), "203.0.113.9:5000", "10.0.0.1")
	if got["resolved_ip"] == "10.0.0.1" {
		t.Error("a forged X-Forwarded-For was believed from an untrusted peer")
	}
	if got["resolved_ip"] != "203.0.113.9" {
		t.Errorf("resolved to %s, want the real peer 203.0.113.9", got["resolved_ip"])
	}
}

// RealIP writes only the context; r.RemoteAddr keeps the peer. Code that reads
// RemoteAddr believing it was normalized is the exact mistake that keyed a
// limit on the proxy - so the promise is pinned.
func TestRemoteAddrIsLeftAlone(t *testing.T) {
	got := behindProxy(router(true), "203.0.113.1")
	if got["resolved_ip"] != "203.0.113.1" {
		t.Errorf("resolved_ip = %s, want the forwarded client", got["resolved_ip"])
	}
	if got["remote_addr"] != "127.0.0.1:9999" {
		t.Errorf("remote_addr = %s, want the proxy peer untouched", got["remote_addr"])
	}
}

// KeyByIP drops the port: every connection from one client has a different
// port, so a key that kept it would give each request its own bucket.
func TestKeyByIPIsPortless(t *testing.T) {
	got := behindProxy(router(true), "203.0.113.1")
	if strings.Contains(got["rate_limit_key"], ":") {
		t.Errorf("key %q still carries a port", got["rate_limit_key"])
	}
	_ = middlewares.KeyByIP // referenced for the doc's sake
}

// The recipe narrative runs clean.
func TestRunPlaysThrough(t *testing.T) {
	var buf bytes.Buffer
	if err := run(&buf); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"SAME key", "separate keys", "RemoteAddr"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("output missing %q:\n%s", want, buf.String())
		}
	}
}
