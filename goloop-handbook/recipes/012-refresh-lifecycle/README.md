# Recipe 012: the refresh-token lifecycle

Part of the [GoLoop One handbook](../../main.md). Read the chapter:
[12. The refresh-token lifecycle](../../chapters/12-refresh-lifecycle.md).

```sh
go test ./...   # runs the auth/authtest conformance suite against the store
go run .        # walks through rotate, reuse, grace, expiry and sign-out
```

No infrastructure is needed: the recipe uses `auth.NewMemoryRefreshStore`. A
production service swaps in a store backed by Redis or Postgres - the chapter
shows the atomic Redis rotation - and `auth/authtest` holds any store to the
same contract.

## The state machine

| Presented token | Status | What it means | Respond with |
|---|---|---|---|
| current | `Rotated` | honest refresh | the new token pair |
| just-rotated, within the window | `PreviousWithinGrace` | a dropped response, not theft | `401`, do **not** revoke |
| older or already revoked | `ReusedStale` | a replayed token | revoke the whole family |
| past its lifetime | `ErrRefreshExpired` | an idle client | `401`; not reuse, do not revoke |

The one rule that ties it together: reuse of a retired token means the family
is compromised, so end it with `RevokeAll`. Everything else is either an
honest refresh or an honest mistake.
