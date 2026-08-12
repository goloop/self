// Recipe 013: behind a reverse proxy, without the silent trap.
//
// Almost every real deployment sits behind nginx, a load balancer or a CDN. To
// the Go process, the connection then comes from the proxy, not the client, so
// r.RemoteAddr is the proxy's address - the same one for every visitor. Anything
// keyed by client address inherits that: a per-IP rate limit becomes one shared
// allowance for the whole userbase, an audit log attributes everything to the
// proxy, a ban list bans everyone or no one. Nothing errors. It just quietly
// stops being per-client, and it does so only in production, because a developer
// hitting the service directly never goes through a proxy.
//
// goloop/middlewares.RealIP resolves the real client address from the forwarded
// headers - but only from proxies you have declared trusted, because a header
// anyone can forge is not evidence. This recipe shows the trap and the fix side
// by side:
//
//	A. the collapse   - without WithTrustedProxies, two clients behind a proxy
//	                    resolve to one address, so one rate-limit bucket serves
//	                    both;
//	B. the fix        - WithTrustedProxies names the proxy, X-Forwarded-For is
//	                    believed from it, and the two clients get separate
//	                    buckets;
//	C. the forgery    - a client that forges X-Forwarded-For directly (no
//	                    trusted proxy in front) is not believed;
//	D. RemoteAddr     - RealIP writes only the context; r.RemoteAddr is left
//	                    untouched, so code that reads it still gets the peer -
//	                    which is why you ask RealIPFrom, never RemoteAddr;
//	E. correlated log - WithContextLogger puts a request-scoped logger carrying
//	                    the request id and client ip into the context, so a
//	                    handler's log lines join up with the request line.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/goloop/middlewares"
	"github.com/goloop/mux"
	"github.com/goloop/resp/v2"
)

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "recipe:", err)
		os.Exit(1)
	}
}

// keyProbe is a handler that reports the rate-limit key the request would use,
// so the demo can show the collapse without waiting for a limit to trip.
func keyProbe(w http.ResponseWriter, r *http.Request) {
	_ = resp.JSON(w, map[string]string{
		"rate_limit_key": middlewares.KeyByIP()(r),
		"remote_addr":    r.RemoteAddr,
		"resolved_ip":    middlewares.RealIPFrom(r.Context()),
	})
}

// router builds the middleware chain. trustProxy toggles the one line that is
// the whole difference between a per-client limit and a shared one.
func router(trustProxy bool) http.Handler {
	realIP := middlewares.RealIP()
	if trustProxy {
		// 127.0.0.0/8 stands in for the address nginx connects from. In a
		// real deployment this is the proxy's own subnet.
		realIP = middlewares.RealIP(
			middlewares.WithTrustedProxies("127.0.0.0/8"))
	}

	r := mux.New()
	r.Use(middlewares.RequestID())
	r.Use(realIP)
	r.Get("/whoami", keyProbe)
	return r
}

// behindProxy makes a request as it arrives after nginx: the peer is the
// proxy (127.0.0.1), and the real client is in X-Forwarded-For.
func behindProxy(h http.Handler, clientIP string) map[string]string {
	return behindProxyFrom(h, "127.0.0.1:9999", clientIP)
}

// behindProxyFrom is behindProxy with the peer address spelled out, so a test
// can send a request whose peer is not a trusted proxy.
func behindProxyFrom(h http.Handler, peer, forwarded string) map[string]string {
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.RemoteAddr = peer
	req.Header.Set("X-Forwarded-For", forwarded)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return out
}

func run(w io.Writer) error {
	fmt.Fprintln(w, "A. behind a proxy, WITHOUT a trusted-proxy policy:")
	naive := router(false)
	c1 := behindProxy(naive, "203.0.113.1")
	c2 := behindProxy(naive, "203.0.113.2")
	fmt.Fprintf(w, "   client 203.0.113.1 -> key %s\n", c1["rate_limit_key"])
	fmt.Fprintf(w, "   client 203.0.113.2 -> key %s\n", c2["rate_limit_key"])
	if c1["rate_limit_key"] == c2["rate_limit_key"] {
		fmt.Fprintln(w, "   ^ SAME key: one rate-limit bucket for the whole userbase")
	}

	fmt.Fprintln(w, "B. WITH WithTrustedProxies(\"127.0.0.0/8\"):")
	fixed := router(true)
	c1 = behindProxy(fixed, "203.0.113.1")
	c2 = behindProxy(fixed, "203.0.113.2")
	fmt.Fprintf(w, "   client 203.0.113.1 -> key %s\n", c1["rate_limit_key"])
	fmt.Fprintf(w, "   client 203.0.113.2 -> key %s\n", c2["rate_limit_key"])
	if c1["rate_limit_key"] != c2["rate_limit_key"] {
		fmt.Fprintln(w, "   ^ separate keys: the limit is per client again")
	}

	fmt.Fprintln(w, "C. a client forging X-Forwarded-For with no trusted proxy in front:")
	forged := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	forged.RemoteAddr = "203.0.113.9:5000"           // the client connects directly
	forged.Header.Set("X-Forwarded-For", "10.0.0.1") // and lies
	rec := httptest.NewRecorder()
	router(true).ServeHTTP(rec, forged)
	var out map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	fmt.Fprintf(w, "   claimed 10.0.0.1, resolved to %s (the real peer, not the lie)\n",
		out["resolved_ip"])

	fmt.Fprintln(w, "D. RealIP leaves r.RemoteAddr untouched:")
	probe := behindProxy(fixed, "203.0.113.1")
	fmt.Fprintf(w, "   RealIPFrom(ctx) = %s   r.RemoteAddr = %s\n",
		probe["resolved_ip"], probe["remote_addr"])
	fmt.Fprintln(w, "   ask RealIPFrom for the client; RemoteAddr is still the proxy")

	fmt.Fprintln(w, "E. a handler's logs correlate with the request:")
	return demoContextLogger(w)
}

// demoContextLogger shows WithContextLogger: the handler logs with the logger
// from the context, and the line carries request_id and remote_ip without the
// handler adding them.
func demoContextLogger(w io.Writer) error {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, nil))

	h := mux.New()
	h.Use(middlewares.RequestID())
	h.Use(middlewares.RealIP(middlewares.WithTrustedProxies("127.0.0.0/8")))
	h.Use(middlewares.Logger(middlewares.WithLogger(base),
		middlewares.WithContextLogger()))
	h.Get("/work", func(w http.ResponseWriter, r *http.Request) {
		// An ordinary log call - no identifiers threaded in by hand.
		middlewares.LoggerFrom(r.Context()).Warn("upstream slow", "ms", 512)
		_ = resp.JSON(w, map[string]string{"ok": "true"})
	})

	req := httptest.NewRequest(http.MethodGet, "/work", nil)
	req.RemoteAddr = "127.0.0.1:9999"
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	h.ServeHTTP(httptest.NewRecorder(), req)

	// Find the handler's own line and show it carries the correlation fields.
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) != nil || m["msg"] != "upstream slow" {
			continue
		}
		fmt.Fprintf(w, "   handler log: msg=%q remote_ip=%v request_id set=%v\n",
			m["msg"], m["remote_ip"], m["request_id"] != nil)
	}
	return nil
}
