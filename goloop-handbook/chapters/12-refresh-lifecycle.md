[« A hardened sign-in](11-hardened-signin.md) · [Contents](../main.md) · [Behind a reverse proxy »](13-reverse-proxy.md)

---

# 12. The refresh-token lifecycle

**Task.** Keep a user signed in without keeping a long-lived credential lying
around. An access token is short-lived on purpose; a refresh token is how the
client gets the next one silently. That makes the refresh token the valuable
thing to steal, so the whole lifecycle turns on one question: when a token that
is no longer current comes back, is this an honest client repeating a request
whose answer it never received, or an attacker replaying a token they took?

**Modules.** [`auth`](https://github.com/goloop/auth) defines the refresh-token
lifecycle - rotation, reuse detection, a grace window, expiry, and "sign out
everywhere" - and [`auth/authtest`](https://github.com/goloop/auth) proves any
store implements the contract correctly, concurrency included.

**Recipe.** [`recipes/012-refresh-lifecycle`](../recipes/012-refresh-lifecycle/)

## The one rule

Every refresh **rotates**: it issues a new token and retires the old one.
Presenting a retired token is therefore a signal, and how you read it is the
whole security of the scheme:

| Presented token | Status | What it means | Respond with |
|---|---|---|---|
| current | `Rotated` | an honest refresh | the new token pair |
| just-rotated, within the window | `PreviousWithinGrace` | a dropped response, not theft | `401`, do **not** revoke |
| older, or already revoked | `ReusedStale` | a replayed token | revoke the whole family |
| past its lifetime | `ErrRefreshExpired` | an idle client | `401`; not reuse |

The rule that ties it together: reuse of a retired token means the token family
is compromised, so end it. Everything else is either an honest refresh or an
honest mistake, and must not sign the user out of their other devices.

## Example A - rotate

`RotateWithStatus` swaps the presented token for a successor and reports what
the attempt was:

```go
next, _, _ := auth.NewRefreshToken(subject, ttl)
res, err := auth.RotateWithStatus(ctx, store, presentedID, next)
// res.Status == auth.Rotated, err == nil: hand `next` to the client
```

## Example B - detect reuse

The same token presented after it was rotated is the theft signal. The response
is not a 401 - it is revoking every session the subject has, so the stolen token
and the legitimate one both die and the user signs in again:

```go
res, err := auth.RotateWithStatus(ctx, store, oldID, next)
if res.Status == auth.ReusedStale {
	auth.RevokeAll(ctx, store, subject) // end the whole family
}
```

A note the recipe is careful about: `RotateResult.Subject` is filled where the
store can still name the owner, but a stale token whose record is long gone
cannot always do so. A real `/refresh` handler has already loaded the record and
verified the secret before it rotates, so it knows the subject from that step -
hold onto it there, rather than depending on the result.

## Example C - the grace window

A client whose connection dropped never saw the successor it was issued, and
retries the token it still holds - the one that was rotated a moment ago. That
is not theft, and punishing it logs honest users out. `auth.WithGrace` opens a
short window in which the immediately previous token is `PreviousWithinGrace`
instead of `ReusedStale`:

```go
store := auth.NewMemoryRefreshStore(auth.WithGrace(10 * time.Second))
// ...replaying the just-rotated token within 10s:
// res.Status == auth.PreviousWithinGrace -> answer 401, do NOT revoke
```

Keep the window short. Every second of it is a second in which a stolen token
would also be accepted once - and it accepts only the *immediately* previous
token, never one two rotations back, which can only be a replay.

## Example D - expiry is not reuse

A token past its lifetime is refused, but the refusal is `ErrRefreshExpired`,
not the reuse signal. This distinction is load-bearing: an idle client whose
token simply aged out is not an attacker, and answering with `ErrRefreshUsed`
would revoke their other sessions for the crime of being away:

```go
_, err := auth.RotateWithStatus(ctx, store, expiredID, next)
errors.Is(err, auth.ErrRefreshExpired) // true
errors.Is(err, auth.ErrRefreshUsed)    // false - do not revoke the family
```

## Example E - the conformance test

The recipe stores tokens in `auth.NewMemoryRefreshStore`, so it runs with zero
infrastructure. A production service puts this behind Redis or Postgres - and
that is where the subtle bugs live: a rotation that is not atomic lets two
clients both get a successor; an index maintained in `Save` and `Revoke` but not
`Rotate` leaves "sign out everywhere" reporting success while a session lives on.

`auth/authtest` holds any store to the whole contract, concurrency included:

```go
func TestRedisStore(t *testing.T) {
	authtest.RefreshStore(t, func(t *testing.T) auth.RefreshStore {
		return newRedisStore(t) // fresh, empty
	})
}
```

It found a real bug in the reference store on its first run. Write your Redis or
Postgres store, point this at it, and the same suite proves it obeys the same
rules - including the ones that pass a single sequential test and fail under
load.

## A shared store, in outline

The reference memory store is single-process. The production shape is in the
[`auth` reference](https://github.com/goloop/auth) under "Writing a shared
store"; the one invariant to carry away is that rotation must be **atomic**. In
Redis that is a script, not a sequence of commands, and the delete is both the
check and the claim:

```lua
if redis.call('del', KEYS[1]) == 1 then   -- whoever deletes the key wins
  redis.call('set', KEYS[2], ARGV[1], 'EX', ARGV[4])  -- store the successor
  -- ...maintain the subject index here too...
  return 1
end
return 0                                    -- someone else already rotated it
```

A `get` then a `del` as two commands leaves a window where both racers pass the
check - which is exactly the race an atomic rotation closes.

## Execution report

```
$ go run .
A. a normal refresh issues the next token:
   status=rotated  a successor was issued
B. replaying the retired token is a theft signal:
   status=reused_stale  err=auth: refresh token already used
   -> RevokeAll("ada"): the whole family is revoked
C. a grace window absorbs a dropped response:
   replaying the just-rotated token: status=previous_within_grace
   -> answer 401, but do NOT revoke: this is a lost response, not theft
D. an expired token is refused, but is not reuse:
   err is ErrRefreshExpired: true (and NOT ErrRefreshUsed: false)
E. sign out everywhere:
   -> every one of dave's refresh tokens is now revoked
```

## What you learned

- Every refresh rotates; a retired token coming back is a signal, read through
  `RotateWithStatus`.
- `ReusedStale` means the family is compromised - answer with `RevokeAll`, not a
  bare 401.
- A grace window (`auth.WithGrace`) tells a dropped response from a replay, so
  honest clients are not signed out; keep it short and it covers only the
  immediately previous token.
- Expiry is `ErrRefreshExpired`, never reuse - an idle client is not a thief.
- Store tokens where you like, but prove the store with `auth/authtest`:
  rotation must be atomic, and the subject index must be maintained in `Save`,
  `Rotate` and `Revoke` alike.

Next: making all of this per-client when a reverse proxy stands in the way.

---

[« A hardened sign-in](11-hardened-signin.md) · [Contents](../main.md) · [Behind a reverse proxy »](13-reverse-proxy.md)
