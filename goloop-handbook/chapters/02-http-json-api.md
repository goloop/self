[« Configuration](01-configuration.md) · [Contents](../main.md) · [Validate and clean »](03-validate-and-clean.md)

---

# 02. A JSON HTTP API without a framework

**Task.** Serve JSON over HTTP: route requests, read path parameters, return
the right status codes, and wrap everything in the usual cross-cutting concerns
(a request id, panic recovery, logging, security headers) - without adopting a
web framework.

**Modules.** [`mux`](https://github.com/goloop/mux) routes (a thin layer over
`net/http.ServeMux`), [`resp`](https://github.com/goloop/resp) writes JSON and
error responses, [`middlewares`](https://github.com/goloop/middlewares) is the
chain of `func(http.Handler) http.Handler` wrappers.

**Recipe.** [`recipes/002-http-json-api`](../recipes/002-http-json-api/)

## The idea

Go's `net/http` already routes, and since Go 1.22 its `ServeMux` understands
method and wildcard patterns like `GET /users/{id}`. What it does not give you
is the small ergonomics on top: a one-liner to write JSON, a clean error body,
and a tidy way to stack middleware. GoLoop adds exactly those, and nothing that
hides `net/http` underneath.

Routing with `mux` reads like the standard patterns, because they *are* the
standard patterns:

```go
r := mux.New()

r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
	_ = resp.JSON(w, resp.R{"status": "ok"})
})

r.Get("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
	u, ok := users[atoi(mux.Param(req, "id"))]
	if !ok {
		_ = resp.Error(w, http.StatusNotFound, "user not found")
		return
	}
	_ = resp.JSON(w, u)
})
```

`resp.JSON(w, v)` encodes and writes; `resp.R` is a shorthand `map[string]any`
for ad-hoc objects; `resp.Error(w, status, message)` writes a consistent error
body. `mux.Param(req, "id")` reads the wildcard. The `Router` is itself an
`http.Handler`.

## The middleware chain

Cross-cutting concerns are plain wrappers, applied outermost-first:

```go
return middlewares.Handler(r,
	middlewares.RequestID(),       // tag each request with an id
	middlewares.Recoverer(),       // turn a panic into a 500, not a crash
	middlewares.Logger(),          // one structured log line per request
	middlewares.SecurityHeaders(), // nosniff, frame options, and friends
)
```

Because each middleware is just `func(http.Handler) http.Handler`, your own fit
in the same list, and nothing here is GoLoop-specific plumbing.

## Execution report

Tested with `httptest` (no network), then deployed and hit with `curl`:

```text
$ go test ./...
INFO http request method=GET path=/health   status=200 bytes=16 ...
INFO http request method=GET path=/users/1   status=200 bytes=48 ...
INFO http request method=GET path=/users/999 status=404 bytes=40 ...
ok  	goloop.one/handbook/002-http-json-api	0.003s

$ ./api &                        # deployed on :8081

$ curl -i localhost:8081/health
HTTP/1.1 200 OK
X-Content-Type-Options: nosniff
X-Request-Id: 53941dbe3ef91b6ca0c52360342fc109
{"status":"ok"}

$ curl localhost:8081/users/1
{"id":1,"name":"Ada","email":"ada@example.com"}

$ curl -w 'HTTP %{http_code}\n' localhost:8081/users/999
{"code":404,"message":"user not found"}
HTTP 404
```

Every response carries the security header and a request id, the path parameter
resolves, and the missing user returns a clean `404` JSON body - the tests
assert all three, and the live `curl` confirms them.

## What you learned

- `mux` routes with the standard `GET /path/{id}` patterns; the router is an
  `http.Handler`.
- `resp.JSON`, `resp.R` and `resp.Error` cover the JSON and error responses you
  write over and over.
- `middlewares.Handler(h, ...)` stacks concerns as ordinary `net/http`
  wrappers; your own middleware drops into the same list.
- Test handlers with `httptest`; the recipe does both that and a live `curl`.

Next: make sure the data coming *in* is clean before you trust it.

---

[« Configuration](01-configuration.md) · [Contents](../main.md) · [Validate and clean »](03-validate-and-clean.md)
