[« Configuration](01-configuration.md) · [Contents](../main.md) · [Validate and clean »](03-validate-and-clean.md)

---

# 02. A JSON HTTP API without a framework

**Task.** Serve JSON over HTTP: route requests, read path parameters and query
parameters, return the right status codes for reads and writes, and wrap
everything in the usual cross-cutting concerns (a request id, panic recovery,
logging, security headers) - plus one of your own - without adopting a web
framework.

**Modules.** [`mux`](https://github.com/goloop/mux) routes (a thin layer over
`net/http.ServeMux`), [`resp`](https://github.com/goloop/resp) writes JSON and
error responses, [`qp`](https://github.com/goloop/qp) reads and validates query
parameters, [`middlewares`](https://github.com/goloop/middlewares) is the chain
of `func(http.Handler) http.Handler` wrappers.

**Recipe.** [`recipes/002-http-json-api`](../recipes/002-http-json-api/)

## Example A - routing and JSON

Go's `net/http` already routes, and since Go 1.22 its `ServeMux` understands
`GET /users/{id}` patterns. `mux` adds the small ergonomics on top; `resp`
writes the JSON:

```go
r := mux.New()

r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
	_ = resp.JSON(w, resp.R{"status": "ok"})
})

r.Get("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
	u, ok := s.get(atoi(mux.Param(req, "id")))
	if !ok {
		_ = resp.Error(w, http.StatusNotFound, "user not found")
		return
	}
	_ = resp.JSON(w, u)
})
```

`resp.JSON(w, v)` encodes and writes; `resp.R` is a shorthand `map[string]any`;
`resp.Error(w, status, message)` writes a consistent error body;
`mux.Param(req, "id")` reads the wildcard. The `Router` is itself an
`http.Handler`.

## Example B - the right status for a write

A read is a `200`. A create should be a `201` with a `Location`; a delete should
be a `204` with an empty body. `resp` has both:

```go
r.Post("/users", func(w http.ResponseWriter, req *http.Request) {
	var in user
	if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
		_ = resp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	u := s.add(in)
	_ = resp.Created(w, "/users/"+strconv.Itoa(u.ID), u) // 201 + Location
})

r.Delete("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
	if !s.del(atoi(mux.Param(req, "id"))) {
		_ = resp.Error(w, http.StatusNotFound, "user not found")
		return
	}
	_ = resp.NoContent(w) // 204, empty body
})
```

## Example C - your own middleware in the chain

Cross-cutting concerns are plain wrappers, applied outermost-first. A custom one
needs no special interface - it is just `func(http.Handler) http.Handler`:

```go
func apiVersion(v string) middlewares.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-API-Version", v)
			next.ServeHTTP(w, r)
		})
	}
}

return middlewares.Handler(r,
	middlewares.RequestID(),
	middlewares.Recoverer(),
	middlewares.Logger(),
	middlewares.SecurityHeaders(),
	apiVersion("v1"), // yours, in the same list
)
```

## Example D - query parameters, read and validated

A list endpoint needs paging and a filter, and those come in the query string:
`GET /users?limit=2&offset=0&q=ada`. `qp` reads each parameter with a type, a
default, and a constraint, so a missing or malformed value never reaches your
handler:

```go
r.Get("/users", func(w http.ResponseWriter, req *http.Request) {
	q := qp.New(req.URL)
	limit := q.Int("limit", qp.Default(20), qp.Between(1, 100))
	offset := q.Int("offset", qp.Default(0), qp.Min(0))
	name := q.String("q")
	_ = resp.JSON(w, resp.R{
		"users":  s.list(limit, offset, name),
		"limit":  limit,
		"offset": offset,
	})
})
```

`qp.Default(v)` is the fallback when the parameter is absent, empty, or invalid.
`qp.Between(1, 100)` and `qp.Min(0)` are constraints: a value outside the range
is *rejected*, not clamped - `qp` drops it and returns the default. So
`limit=999` does not become `100`; it becomes the default `20`. Your handler
receives a value it can trust and never has to re-check the bounds.

## Execution report

Tested with `httptest`, then deployed and hit with `curl`:

```text
$ go test ./...
ok  	goloop.one/handbook/002-http-json-api	0.004s

$ ./api &                        # deployed on :8081

$ curl -D - -o /dev/null localhost:8081/health
HTTP/1.1 200 OK
X-Api-Version: v1
X-Content-Type-Options: nosniff
X-Request-Id: 5fdbdc2b60e7ee66b91294605c3e0a2b

$ curl -D - -X POST localhost:8081/users -d '{"name":"Grace","email":"grace@example.com"}'
HTTP/1.1 201 Created
Location: /users/2
{"id":2,"name":"Grace","email":"grace@example.com"}

$ curl -w 'HTTP %{http_code} bytes=%{size_download}\n' -X DELETE localhost:8081/users/1
HTTP 204 bytes=0

$ curl -w 'HTTP %{http_code}\n' localhost:8081/users/1   # after the delete
HTTP 404

$ curl -s 'localhost:8081/users?limit=2'
{"limit":2,"offset":0,"users":[{"id":1,...},{"id":2,...}]}

$ curl -s 'localhost:8081/users?q=gra'                   # name filter
{"limit":20,"offset":0,"users":[{"id":2,"name":"Grace",...}]}

$ curl -s 'localhost:8081/users?limit=999'               # out of range
{"limit":20,...}                                         # rejected -> default 20
```

Every response carries the security header, the request id and your
`X-Api-Version`. The create returns `201` with `Location: /users/2`; the delete
returns `204` with no body; and the deleted user is then `404`. The list honors
`limit=2` and the `q` filter, and rejects `limit=999` back to the default `20`
rather than clamping it.

## What you learned

- `mux` routes with the standard `GET /path/{id}` patterns; the router is an
  `http.Handler`.
- `resp.JSON`/`resp.R`/`resp.Error` cover reads; `resp.Created` (201 +
  Location) and `resp.NoContent` (204) cover writes.
- `qp.New(req.URL)` reads query parameters with a type, a `Default`, and
  constraints (`Between`, `Min`); an out-of-range value is rejected and replaced
  by the default, not clamped.
- `middlewares.Handler(h, ...)` stacks concerns; your own middleware is just a
  `func(http.Handler) http.Handler` in the same list.
- Test with `httptest`; the recipe does that and a live `curl`.

Next: make sure the data coming *in* is clean before you trust it.

---

[« Configuration](01-configuration.md) · [Contents](../main.md) · [Validate and clean »](03-validate-and-clean.md)
