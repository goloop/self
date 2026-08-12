# Recipe 011: a hardened sign-in

Part of the [GoLoop One handbook](../../main.md). Read the chapter:
[11. A sign-in endpoint that survives production](../../chapters/11-hardened-signin.md).

```sh
go test ./...   # includes the wiring test that proves the rate limit is mounted
go run .        # tells the story against the recipe's own router
```

No environment is required to run it: an ephemeral JWT secret is generated when
`JWT_SECRET` is absent (fine for a demo, wrong for production - see below).

## Environment

| Variable | Default | Meaning |
|---|---|---|
| `JWT_SECRET` | *(ephemeral)* | access-token signing key; **at least 32 bytes**, `required` in production |
| `AUTH_ATTEMPTS` | `10` | sign-in attempts per address per window |
| `AUTH_WINDOW` | `15m` | the window that budget is spent over |
| `AUTH_CONCURRENCY` | `32` | requests to `/auth/*` served at once (edge shedding) |
| `HASH_CONCURRENCY` | `4` | simultaneous Argon2id operations; **memory budget = this x cost-per-hash** |

Copy `.env.example` to `.env` and fill it, or export the variables. In
production set a stable `JWT_SECRET` and change the tag to
`env:"JWT_SECRET,required"`, so a missing secret stops the deploy at boot
instead of the first login.

## The pattern in one line

Identify the caller, refuse the too-frequent, shed the too-many, spend the same
cost whether or not the account exists, and answer every refusal with a slug -
and prove with a test that the brakes are actually mounted, not just written.
