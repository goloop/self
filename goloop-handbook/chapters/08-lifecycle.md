[« Slugs and cases](07-slug.md) · [Contents](../main.md) · [Real-time with WebSockets »](09-websocket.md)

---

# 08. A service lifecycle

**Task.** Run an HTTP server as a real service: start it, expose liveness and
readiness probes, log in a structured way, and shut it down cleanly when a
signal arrives - stopping components in order, not leaving connections hanging.

**Modules.** [`app`](https://github.com/goloop/app) is the ordered start/stop
lifecycle, [`observe`](https://github.com/goloop/observe) provides `/healthz`
and `/readyz`, and [`log`](https://github.com/goloop/log) gives a shared
structured `slog` logger.

**Recipe.** [`recipes/008-lifecycle`](../recipes/008-lifecycle/)

## Example A - the lifecycle

`app` runs a set of components with an ordered start, a signal-aware wait and a
graceful reverse shutdown. An `http.Server` is a ready-made component:

```go
a := app.New("handbook-svc", app.WithLogger(logger))
a.Use(app.HTTPServer("http", &http.Server{Addr: ":8085", Handler: mux}))
a.OnStart(func(context.Context) error { logger.Info("service up"); return nil })
a.OnStop(func(context.Context) error { logger.Info("service stopping"); return nil })
return a.Run(ctx) // blocks until ctx is cancelled or SIGINT/SIGTERM, then stops cleanly
```

`Run` returns `nil` on a clean shutdown, so a `main` that returns its error does
the right thing on both a signal and a fatal component error.

## Example B - liveness and readiness

`observe` separates two questions. Liveness (`/healthz`) asks "is the process
up?" and is always ok. Readiness (`/readyz`) asks "are dependencies usable?" and
runs every registered check:

```go
reg := observe.New(observe.WithService("handbook-svc"),
	observe.WithBuildInfo(observe.BuildInfo{Version: "1.0.0"}))
reg.Check("clock", func(context.Context) error { return nil }) // your real check here

mux.Handle("GET /healthz", reg.HealthHandler())
mux.Handle("GET /readyz", reg.ReadyHandler())
```

## Example C - one shared logger

`log.NewSlog` returns a standard `*slog.Logger`, so every part of the service -
your code and `app` itself - logs through the same structured logger:

```go
logger := log.NewSlog("svc")
```

## Execution report

Tested (including a real start-and-shutdown), then deployed, probed, and sent a
signal (log paths trimmed):

```text
$ go test ./...
ok  	goloop.one/handbook/008-lifecycle	0.307s

$ ./svc &                        # serves on :8085

$ curl localhost:8085/healthz
{"status":"ok","service":"handbook-svc","version":"1.0.0"}

$ curl localhost:8085/readyz
{"status":"ok","service":"handbook-svc","version":"1.0.0",
 "checks":{"clock":{"status":"ok","latency_ms":0}}}

$ kill -INT %1                   # graceful shutdown
INFO service up addr=:8085
INFO starting component component=http
INFO running components=1
INFO shutdown signal received
INFO stopping component component=http
INFO service stopping
```

Liveness answered immediately; readiness ran the `clock` check and reported it;
and the signal walked the lifecycle backward - stop the component, run the stop
hook - instead of dropping the process.

## What you learned

- `app` runs components with an ordered start and a graceful reverse shutdown;
  `app.HTTPServer` wraps an `http.Server`, and `Run(ctx)` blocks until a signal
  or cancellation.
- `observe` splits liveness (`/healthz`, always ok) from readiness (`/readyz`,
  runs your `Check` functions), the distinction orchestrators expect.
- `log.NewSlog` gives one structured `*slog.Logger` for the whole service,
  including `app`'s own lifecycle logs.

Next: keep a connection open for real-time updates.

---

[« Slugs and cases](07-slug.md) · [Contents](../main.md) · [Real-time with WebSockets »](09-websocket.md)
