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

## Execution report

```text
$ go test ./...
ok  	goloop.one/handbook/006-auth	0.297s

$ go run .
A. password hash and verify (auth PBKDF2):
   stored hash: pbkdf2-sha256$600000$4sLgsWRnhEQ... (87 bytes, no plaintext)
   verify correct password: true
   verify wrong password:   false
B. access token issue and verify (auth + jwt), key public id:
   token (JWT): eyJhbGciOiJIUzI1NiIsInR5cCI6IkpX...
   verified subject: id=42 email=ada@example.com roles=[admin]
   tampered token rejected: true
   key public id: qj23hctskd
C. signed session cookie (session):
   set cookie session (v1.eyJpZCI6ImJjYTBkODQyY...)
   loaded session: subject=42 theme=dark
```

The hash keeps no plaintext and the wrong password fails; the JWT round-trips
its subject and a single altered character is rejected; and the session survives
a round trip through a cookie.

## What you learned

- `auth.NewPBKDF2` hashes and verifies passwords; the digest carries its own
  algorithm, cost and salt, so `Verify` needs nothing else.
- `auth.TokenManager` issues and verifies signed tokens (JWT); a `Subject`
  carries the id, email, roles and scopes. Protect routes with its `Protect`
  middleware (see the whole-stack chapter).
- `session` keeps a signed cookie session with `Save`/`Load`; `key` mints short,
  unambiguous public ids.
- None of this stores a secret you could leak: hashes, not passwords; signatures,
  not trust.

Next: readable URLs from any text.

---

[« Language models](05-ai.md) · [Contents](../main.md) · [Slugs and cases »](07-slug.md)
