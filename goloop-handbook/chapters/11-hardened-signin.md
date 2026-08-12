[« Putting it together](10-whole-stack.md) · [Contents](../main.md) · [The refresh-token lifecycle »](12-refresh-lifecycle.md)

---

# 11. A sign-in endpoint that survives production

**Task.** Take the sign-in handler that works on day one and make it survive
day one hundred. The naive version fails three ways in production, none of them
visible in development: a missing signing secret that only surfaces when the
first user tries to sign in; a password-guessing script that hits the endpoint
a thousand times a minute; and a burst of parallel attempts that exhausts
memory, because every Argon2id verification - right password or wrong - costs
tens of megabytes by design.

**Modules.** [`env`](https://github.com/goloop/env) reads and validates
configuration, [`jwt`](https://github.com/goloop/jwt) checks the signing key,
[`argon2id`](https://github.com/goloop/argon2id) hashes passwords,
[`auth`](https://github.com/goloop/auth) issues tokens and levels the timing,
[`middlewares`](https://github.com/goloop/middlewares) rate-limits and throttles,
[`mux`](https://github.com/goloop/mux) groups the routes, and
[`resp`](https://github.com/goloop/resp) answers with a machine-readable slug.

**Recipe.** [`recipes/011-hardened-signin`](../recipes/011-hardened-signin/)

This is the first chapter of Part IV, which builds patterns rather than
introducing modules. Each one is drawn from services running the whole GoLoop
stack in production, and each recipe is a complete program you can run.

## The pattern

Four brakes in front of one expensive operation, in this order:

1. **Fail at boot, not at first login.** A missing or weak signing secret is a
   configuration error. Caught at startup, it is a rolled-back deploy; caught
   at the first sign-in, it is an outage.
2. **Rate-limit the guesser.** A per-address budget per window stops a script
   from trying millions of passwords, and never inconveniences a person who
   fumbles theirs a few times.
3. **Shed the burst.** A concurrency limit answers a different threat from the
   rate limit: not "too often" but "too many at once", which is what exhausts
   memory in front of a memory-hard hash.
4. **Cost the same either way.** The "no such account" branch must spend the
   same hashing work as "wrong password", or response time reads out the user
   table.

And, running through all of it, one rule for errors: every refusal is the same
status with the same machine-readable slug, so the frontend can act on it and
an attacker learns nothing from it.

## Example A - fail at boot

`env` fills the config from tags; `Validate` checks what tags cannot. The
load-bearing line is `jwt.CheckKey`, the same rule `Sign` and `Verify` apply on
every call, exported precisely so configuration can fail first:

```go
func (c *Config) Validate() error {
	if err := jwt.CheckKey([]byte(c.JWTSecret)); err != nil {
		return fmt.Errorf("JWT_SECRET is unusable: %w", err)
	}
	// ... cross-field checks ...
	return nil
}
```

`TokenManager.Check` is the same idea for the manager: "can this sign and
verify at all", asked at startup without minting a token. Together they turn
"the first login fails in production" into "the process refuses to start".

## Example B - two brakes, two threats

The rate limit and the throttle look similar and answer opposite questions.
Mounted on the auth group so the rest of the API is not slowed:

```go
r.Route("/auth", func(g *mux.Router) {
	g.Use(middlewares.RateLimit(middlewares.RateLimitConfig{
		Limit:  cfg.AuthAttempts, // e.g. 10 attempts...
		Window: cfg.AuthWindow,   // ...per 15 minutes, per client address
	}))
	g.Use(middlewares.Throttle(cfg.AuthConcurrency, // at most N at once
		middlewares.WithThrottleRetryAfter(2)))
	g.Post("/signin", s.signIn)
})
```

`RateLimit` is "how often may one client come back". `Throttle` is "how much of
this work fits in the process at once". Ten per minute still admits ten in the
same instant; ten concurrent slots do not. In front of sign-in both belong -
and `RateLimit` fails **closed** by default, so a store outage refuses requests
rather than silently removing the limit from your login route.

## Example C - the memory budget

The hasher lives behind a semaphore, and the semaphore lives around the
operation, not on a route:

```go
func (s *service) verify(encoded string, password []byte) error {
	s.guard <- struct{}{}          // one of a fixed number of slots
	defer func() { <-s.guard }()
	return s.hasher.Verify(encoded, password)
}
```

The memory belongs to the process, not to any one endpoint: the hasher is
reachable from sign-in, password change and admin resets alike, so a budget
that lived on a route could be exceeded by going through another. The sizing
rule is the only number to remember: **cost per hash times concurrency must fit
the process's memory budget.** At the Argon2id default of 64 MiB, eight slots is
half a gigabyte. The hasher deliberately does not throttle itself - it cannot
know your budget - which is why the semaphore is yours to place.

## Example D - the same cost for a missing account

`auth.BurnVerify` spends a real verification against a decoy hash on the branch
where there is no user:

```go
stored, ok := s.users[email]
if !ok {
	auth.BurnVerify(s.hasher, s.decoy, []byte(password)) // same work, no user
	// ...answer the same 401 with the same slug as a wrong password
	return
}
```

The decoy is a real hash of anything, built once at startup by the same hasher
with the same settings. Without this, "no such account" answers in a
millisecond and "wrong password" in a hundred, and a stopwatch reads out which
addresses are registered - identical error messages notwithstanding. It levels
this one difference and nothing else: rate limiting and uniform replies are
still yours to add (they are, above).

## Example E - the wiring test

This is the test the chapter is really about, and it is the pattern's quiet
insistence: **prove the protection is mounted, not merely written.** A green
unit test on `RateLimit` says the middleware works. It says nothing about
whether anyone attached it to the route. So the test drives the real router and
asserts the brake bites:

```go
// Fourth attempt from one address, budget of three, must be 429.
var got429 bool
for i := 0; i < cfg.AuthAttempts+2; i++ {
	if post(h, addr, body).Code == http.StatusTooManyRequests {
		got429 = true
		break
	}
}
if !got429 {
	t.Fatal("the rate limit is not attached to /auth: ...")
}
```

The load-bearing detail: comment out `g.Use(middlewares.RateLimit(...))` in the
router and this test fails. A test whose failure nobody has seen is worth no
more than the dead code it guards.

## Execution report

```
$ go run .
note: JWT_SECRET not set; using an ephemeral dev secret
A. boot checks passed: jwt.CheckKey and TokenManager.Check are green
B. sign in:
   right password:  200 {"access_token":"eyJhbGciOiJIUzI1NiIsInR...
   wrong password:  401 {"code":401,"error":"invalid_credentials","message":"Invalid credentials"}
   unknown account: 401 {"code":401,"error":"invalid_credentials","message":"Invalid credentials"}  <- same status, same slug, same cost
C. the rate limit bites after the budget:
   attempt 4: 429 Too Many Requests
D. every refusal carried a slug the frontend can switch on
```

## What you learned

- Validate configuration at boot, and check the signing key with `jwt.CheckKey`
  (and `TokenManager.Check`), so a bad secret stops the deploy instead of the
  first login.
- `middlewares.RateLimit` (fail-closed) and `middlewares.Throttle` answer
  different threats - frequency and concurrency - and a login endpoint wants
  both.
- Bound Argon2id concurrency with a semaphore around the operation, sized so
  cost-per-hash times concurrency fits the process; the hasher will not do it
  for you.
- `auth.BurnVerify` costs the missing-account branch the same as a wrong
  password, so timing does not leak the user table.
- Answer every refusal with the same status and slug (`resp.WithErrorSlug`), and
  test that the brakes are actually mounted - a protection nobody wired up is
  not a protection.

Next: what happens after sign-in - refreshing a session without handing an
attacker a key.

---

[« Putting it together](10-whole-stack.md) · [Contents](../main.md) · [The refresh-token lifecycle »](12-refresh-lifecycle.md)
