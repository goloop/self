[« The refresh-token lifecycle](12-refresh-lifecycle.md) · [Contents](../main.md) · [AI, production shape »](14-ai-production.md)

---

# 13. Behind a reverse proxy

**Task.** Ship a service that sits behind nginx, a load balancer or a CDN -
which is nearly every service - without the silent trap that comes with it. To
the Go process, a request behind a proxy arrives *from the proxy*, so
`r.RemoteAddr` is the proxy's address, the same one for every visitor. Anything
keyed by client address inherits that: a per-IP rate limit becomes one shared
allowance for the whole userbase, an audit log blames the proxy for everything,
a ban list bans everyone or no one. Nothing errors. It just quietly stops being
per-client - and only in production, because a developer hitting the service
directly never goes through a proxy.

**Modules.** [`middlewares`](https://github.com/goloop/middlewares) resolves the
real client IP (`RealIP`), keys a limit by it (`KeyByIP`), and carries a
request-scoped logger (`WithContextLogger` / `LoggerFrom`);
[`mux`](https://github.com/goloop/mux) and
[`resp`](https://github.com/goloop/resp) round out the handler.

**Recipe.** [`recipes/013-reverse-proxy`](../recipes/013-reverse-proxy/)

## The trap, and why the fix is guarded

The real client address is in a forwarded header - `X-Forwarded-For`, set by the
proxy. But a header is not evidence: anyone can send `X-Forwarded-For: 1.2.3.4`
and claim to be someone else. So `RealIP` believes the header **only** from
peers you have declared trusted, and trusts the direct connection otherwise.
That makes the safe default - trust nothing - also the one that silently shares
one bucket across everyone the moment a proxy appears, until you name it.

## Example A - the collapse

Without a trusted-proxy policy, two clients behind the same proxy resolve to the
proxy's address, so they share a rate-limit key:

```go
r.Use(middlewares.RealIP()) // no trusted proxies

// behind nginx, both requests have RemoteAddr = the proxy:
//   client 203.0.113.1 -> key 127.0.0.1
//   client 203.0.113.2 -> key 127.0.0.1   <- SAME key
```

One bucket for the whole userbase. A twenty-attempts-per-address login limit is
now twenty for everyone, and the second person to mistype a password is locked
out by the first.

## Example B - the fix

`WithTrustedProxies` names the proxy's subnet. Now the forwarded header is
believed from it, and the two clients get separate keys again:

```go
r.Use(middlewares.RealIP(
	middlewares.WithTrustedProxies("127.0.0.0/8"))) // your proxy's subnet

//   client 203.0.113.1 -> key 203.0.113.1
//   client 203.0.113.2 -> key 203.0.113.2   <- separate again
```

In a real deployment `127.0.0.0/8` is nginx on the same host; use your load
balancer's range, or your CDN's published ranges.

## Example C - the forgery is not believed

A client that connects directly - no trusted proxy in front - and forges the
header is still resolved to its real address:

```go
// peer 203.0.113.9 (not in the trusted range), header claims 10.0.0.1:
//   resolved to 203.0.113.9  <- the real peer, not the lie
```

This is why the default is safe: the resolved address is trustworthy precisely
because it only believes headers from peers you vouched for.

## Example D - `RemoteAddr` is left untouched

This is the detail a real project got wrong, so it earns its own example.
`RealIP` writes the resolved address into the **request context** - and nothing
else. It never rewrites `r.RemoteAddr`:

```go
middlewares.RealIPFrom(r.Context()) // "203.0.113.1" - the client
r.RemoteAddr                        // "127.0.0.1:9999" - still the proxy
```

Ask `RealIPFrom` for the client address. Code that reads `r.RemoteAddr`
believing the middleware normalized it will key on the proxy - which is the
exact collapse from example A, shipped by a project whose comment said "RealIP
normalized RemoteAddr" while the code did no such thing.

## Example E - logs that correlate

`Logger` writes one line per request with its id. `WithContextLogger` puts a
request-scoped logger - already carrying `request_id`, `method`, `path` and the
resolved `remote_ip` - into the context, so a handler's own log lines join up
with the request without anyone threading identifiers by hand:

```go
r.Use(middlewares.Logger(middlewares.WithContextLogger()))

// in a handler:
middlewares.LoggerFrom(r.Context()).Warn("upstream slow", "ms", 512)
// -> {"msg":"upstream slow","ms":512,"request_id":"...","remote_ip":"203.0.113.7",...}
```

The alternative is remembering to add the id at every call site, which fails
quietly and selectively: the lines that lack it are the ones written in a hurry,
which are the ones written during an incident.

## Execution report

```
$ go run .
A. behind a proxy, WITHOUT a trusted-proxy policy:
   client 203.0.113.1 -> key 127.0.0.1
   client 203.0.113.2 -> key 127.0.0.1
   ^ SAME key: one rate-limit bucket for the whole userbase
B. WITH WithTrustedProxies("127.0.0.0/8"):
   client 203.0.113.1 -> key 203.0.113.1
   client 203.0.113.2 -> key 203.0.113.2
   ^ separate keys: the limit is per client again
C. a client forging X-Forwarded-For with no trusted proxy in front:
   claimed 10.0.0.1, resolved to 203.0.113.9 (the real peer, not the lie)
D. RealIP leaves r.RemoteAddr untouched:
   RealIPFrom(ctx) = 203.0.113.1   r.RemoteAddr = 127.0.0.1:9999
   ask RealIPFrom for the client; RemoteAddr is still the proxy
E. a handler's logs correlate with the request:
   handler log: msg="upstream slow" remote_ip=203.0.113.7 request_id set=true
```

## What you learned

- Behind a proxy, `r.RemoteAddr` is the proxy for every visitor; anything keyed
  by it silently becomes shared.
- `middlewares.RealIP` resolves the real client address, but only believes the
  forwarded header from `WithTrustedProxies` peers - a header anyone can forge is
  not evidence.
- The safe default (trust nothing) is correct only once you name your proxy; set
  `WithTrustedProxies` to its real subnet in every proxied deployment.
- `RealIP` writes the context only. Ask `RealIPFrom(ctx)`, never `r.RemoteAddr`.
- `WithContextLogger` + `LoggerFrom` correlate handler logs with the request by
  default, instead of by everyone remembering.

Next: an AI feature shaped the way production wants it.

---

[« The refresh-token lifecycle](12-refresh-lifecycle.md) · [Contents](../main.md) · [AI, production shape »](14-ai-production.md)
