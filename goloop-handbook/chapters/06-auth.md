[« Language models](05-ai.md) · [Contents](../main.md) · [Slugs and cases »](07-slug.md)

---

# 06. Sessions, tokens and passwords

**Task.** Sign users in safely. Store a password so a database leak does not
hand out plaintext; issue a signed token an API client sends back on each
request; and keep a browser session in a signed cookie. Do not invent any of
this yourself.

**Modules.** [`auth`](https://github.com/goloop/auth) hashes passwords and
issues tokens (a real JWT via [`jwt`](https://github.com/goloop/jwt)),
[`session`](https://github.com/goloop/session) manages a signed cookie session,
and [`key`](https://github.com/goloop/key) mints short public ids.

**Recipe.** [`recipes/006-auth`](../recipes/006-auth/)

## Example A - hash a password

Never store a password. `auth.NewPBKDF2` returns a hasher; `Hash` produces a
salted PBKDF2 digest, and `Verify` checks a candidate in constant time:

```go
hasher := auth.NewPBKDF2()
encoded, _ := hasher.Hash([]byte("correct horse battery staple"))
// encoded == "pbkdf2-sha256$600000$...": algorithm, cost and salt travel with it
ok := hasher.Verify(encoded, []byte("correct horse battery staple")) == nil // true
no := hasher.Verify(encoded, []byte("guess")) == nil                        // false
```

## Example B - issue an access token

`auth.TokenManager` signs a `Subject` into a JWT. `Verify` returns the subject,
and a tampered token fails. `key` gives you a short public id for a URL:

```go
tm := auth.NewTokenManager([]byte(signingSecret), auth.WithIssuer("handbook"))
token, _ := tm.Issue(auth.Subject{ID: "42", Email: "ada@example.com", Roles: []string{"admin"}})
sub, err := tm.Verify(token) // sub.ID == "42", sub.Roles == ["admin"]
_, bad := tm.Verify(token + "x") // bad != nil

pid, _ := key.NewFixed("23456789abcdefghjkmnpqrstuvwxyz", 10)
code, _ := pid.RandomCrypto() // e.g. "qj23hctskd", safe to show
```

## Example C - a session cookie

For a browser, keep the session in a signed cookie. `Save` writes it; a later
request reads it back with `Load`. Here `httptest` stands in for the browser:

```go
mgr := session.New([]byte(sessionKey))
s := mgr.LoadOrNew(req)
s.Subject = "42"
s.Set("theme", "dark")
_ = mgr.Save(w, s) // signs and sets the cookie

// on the next request, with that cookie:
loaded, _ := mgr.Load(req2) // loaded.Subject == "42", loaded.Get("theme") == "dark"
```

## Example D - make the token expire

An access token should be short-lived. `auth.WithTTL` sets the lifetime;
`auth.WithClock` pins "now", so a test can watch it expire without waiting.
Here two managers share the secret but read different clocks: one verifies
inside the window, the other after it:

```go
issuedAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
short := auth.NewTokenManager(secret, auth.WithTTL(30*time.Second),
	auth.WithClock(func() time.Time { return issuedAt }))
tok, _ := short.Issue(auth.Subject{ID: "42"})

early := auth.NewTokenManager(secret,
	auth.WithClock(func() time.Time { return issuedAt.Add(10 * time.Second) }))
late := auth.NewTokenManager(secret,
	auth.WithClock(func() time.Time { return issuedAt.Add(time.Minute) }))

_, errEarly := early.Verify(tok) // nil: still inside the 30s window
_, errLate := late.Verify(tok)   // non-nil: expired
```

In production you pass a real clock (the default) and simply hand out tokens
with a short TTL; the pinned clock is only to show the boundary exactly.

## Example E - a refresh token

A short access token needs a longer-lived way to get a new one.
`auth.NewRefreshToken` returns a record to store and an opaque `"id.secret"`
string for the client. You persist only the hash, never the secret, so a leak
of your store cannot mint tokens. `ParseRefreshToken` splits a returned string,
and `record.Verify(secret)` checks it in constant time:

```go
record, opaque, _ := auth.NewRefreshToken("42", 30*24*time.Hour)
// give `opaque` to the client; store `record` (record.Hash, never the secret).

id, secret, _ := auth.ParseRefreshToken(opaque) // look the record up by id
ok := record.Verify(secret) == nil              // true; a wrong secret is false
```

## Execution report

```text
$ go test ./...
ok  	goloop.one/handbook/006-auth	0.314s

$ go run .
A. password hash and verify (auth PBKDF2):
   stored hash: pbkdf2-sha256$600000$tx8dy2JXhGg... (87 bytes, no plaintext)
   verify correct password: true
   verify wrong password:   false
B. access token issue and verify (auth + jwt), key public id:
   token (JWT): eyJhbGciOiJIUzI1NiIsInR5cCI6IkpX...
   verified subject: id=42 email=ada@example.com roles=[admin]
   tampered token rejected: true
   key public id: x4t8tumhjw
C. signed session cookie (session):
   set cookie session (v1.eyJpZCI6ImVlYzM0MDgxY...)
   loaded session: subject=42 theme=dark
D. token expiry (auth WithTTL + WithClock):
   valid 10s in:  true
   expired at 60s: true
E. refresh token (auth NewRefreshToken / ParseRefreshToken):
   client token: 8ab00e6c4ab95267... (id.secret)
   stored: id=8ab00e6c... hash=86fb21aba9ac... (no secret)
   verify presented secret: true
   verify wrong secret:     false
```

The hash keeps no plaintext and the wrong password fails; the JWT round-trips
its subject and a single altered character is rejected; the session survives a
round trip through a cookie; the short-lived token is valid at 10s and rejected
at 60s; and the refresh token verifies its secret while only its hash is stored.

## What you learned

- `auth.NewPBKDF2` hashes and verifies passwords; the digest carries its own
  algorithm, cost and salt, so `Verify` needs nothing else.
- `auth.TokenManager` issues and verifies signed tokens (JWT); a `Subject`
  carries the id, email, roles and scopes. Protect routes with its `Protect`
  middleware (see the whole-stack chapter).
- `auth.WithTTL` bounds a token's lifetime and `WithClock` makes the expiry
  testable; a longer-lived `auth.NewRefreshToken` mints new ones while you store
  only its hash.
- `session` keeps a signed cookie session with `Save`/`Load`; `key` mints short,
  unambiguous public ids.
- None of this stores a secret you could leak: hashes, not passwords; signatures,
  not trust.

Next: readable URLs from any text.

---

[« Language models](05-ai.md) · [Contents](../main.md) · [Slugs and cases »](07-slug.md)
