# Recipe 013: behind a reverse proxy

Part of the [GoLoop One handbook](../../main.md). Read the chapter:
[13. Behind a reverse proxy](../../chapters/13-reverse-proxy.md).

```sh
go test ./...   # pins the IP collapse, the fix, and that RemoteAddr is untouched
go run .        # shows the trap and the fix side by side
```

No environment is required. The demo uses `httptest` to play both the proxy and
the clients, so you see the resolved keys without deploying anything.

## The trap in one line

Behind a proxy, `r.RemoteAddr` is the proxy for every visitor, so anything keyed
by it - rate limits, audit, ban lists - silently becomes shared. `RealIP` fixes
it, but only once `WithTrustedProxies` tells it which peer is the proxy, because
a forwarded header from anyone else is a forgery, not evidence.

`RealIP` writes the resolved address into the **context** only; it never
rewrites `r.RemoteAddr`. Ask `middlewares.RealIPFrom(ctx)` for the client
address - reading `RemoteAddr` and believing it was normalized is the exact bug
this recipe reproduces.

## Production note

Set `WithTrustedProxies` to your proxy's real subnet (nginx on the same host,
your load balancer's range, or your CDN's published ranges). Leaving it empty is
the safe default against forgery - and the one that silently shares one bucket
across your whole userbase the moment a proxy appears in front.
