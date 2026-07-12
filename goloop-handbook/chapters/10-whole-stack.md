[« Real-time with WebSockets](09-websocket.md) · [Contents](../main.md) · [Contents »](../main.md)

---

# 10. Putting it together - one API on the whole stack

**Task.** Build a small but complete service: a notes API where a person signs
up, logs in, and creates and lists notes behind a bearer token. This chapter
does not introduce a new module; it assembles the ones from the earlier chapters
into one program, so you can see how they compose.

**Modules.** Everything so far: `env`/`opt` (config), `app`/`observe`/`log`
(lifecycle, probes, logging), `mux`/`resp`/`middlewares` (HTTP), `norm`
(validation), `auth` (passwords and tokens), and a `pgc`-generated store
(database).

**Recipe.** [`recipes/010-whole-stack`](../recipes/010-whole-stack/)

## How it fits together

Read `main.go` top to bottom and each chapter reappears in its place:

- **Config** from the environment and flags fills a `Config` struct
  (`env.LoadSafe`, `env.Unmarshal`, `opt.UnmarshalArgs`).
- **Lifecycle**: `app.New` runs an `HTTPServer` component and the `api` itself
  as a component, so the database pool opens in `Start` and closes in `Stop`.
- **Observability**: `observe` serves `/healthz` and `/readyz`, and the
  readiness check pings the database.
- **HTTP**: `mux` routes, `resp` writes JSON and errors, `middlewares` adds the
  request id, recovery, logging and security headers.
- **Validation**: `norm.EmailFold` folds the email into an identity;
  `norm.Clean` scrubs the note title.
- **Auth**: `auth` hashes the password and issues a token; `tm.Protect` guards
  the note routes so an unauthenticated request is a `401`.
- **Database**: the `pgc`-generated `store` gives typed `CreateUser`,
  `UserByEmail`, `CreateNote` and `NotesByUser`.

The protected routes are the crux - one wrapper turns any handler into an
authenticated one:

```go
r.Post("/v1/signup", a.signup)
r.Post("/v1/login", a.login)
r.Handle("POST /v1/notes", a.tm.Protect(http.HandlerFunc(a.createNote)))
r.Handle("GET /v1/notes", a.tm.Protect(http.HandlerFunc(a.listNotes)))
```

## Execution report

Migrated, tested against a real PostgreSQL, then deployed and driven end to end:

```text
$ pgc migrate
applied 001_schema.sql

$ go test ./...            # signup, protected create, 401 without a token
ok  	goloop.one/handbook/010-whole-stack	0.118s

$ ./notes &               # serves on :8087

$ curl localhost:8087/readyz
{"status":"ok","service":"notes","checks":{"database":{"status":"ok","latency_ms":0}}}

$ curl -X POST localhost:8087/v1/signup -d '{"Email":"Ada@Example.com","Password":"password1"}'
201 -> {"token":"eyJhbGci...","email":"ada@example.com"}

$ curl -X POST localhost:8087/v1/notes -d '{"Title":"x"}'          # no token
HTTP 401

$ curl -X POST localhost:8087/v1/notes -H "Authorization: Bearer $TOKEN" -d '{"Title":"Read the handbook"}'
{"id":1,"user_id":1,"title":"Read the handbook","created_at":"..."}

$ curl localhost:8087/v1/login -d '{"Email":"ada@example.com","Password":"password1"}'   # case-insensitive
$ curl localhost:8087/v1/notes -H "Authorization: Bearer $TOKEN"
   notes: ["Read the handbook"]
```

The signup folded `Ada@Example.com` to a lower-case identity and returned a
token; the note route rejected the request with no token and accepted it with
one; login found the user case-insensitively; and readiness confirmed the
database. Every layer from the earlier chapters is doing its one job.

## What you learned

- The modules compose without glue: config fills a struct, `app` runs the
  server and the pool, `observe` reports health, `mux`/`resp`/`middlewares`
  serve HTTP, `auth` protects routes, and the `pgc` store is the database.
- `auth`'s `Protect` turns any handler into an authenticated one; the subject id
  from the token is the user.
- Because each piece is a separate module, this service is a set of choices, not
  a framework you had to accept whole. Swap the database driver, the mail
  transport or the model provider without touching the rest.

That is the whole loop: from configuration in, to a request routed, validated,
authenticated, stored and answered. Build your own from here.

---

[« Real-time with WebSockets](09-websocket.md) · [Contents](../main.md) · [Contents »](../main.md)
