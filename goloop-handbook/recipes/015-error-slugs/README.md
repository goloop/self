# Recipe 015: errors a frontend can switch on

Part of the [GoLoop One handbook](../../main.md). Read the chapter:
[15. Errors a frontend can switch on](../../chapters/15-error-slugs.md).

```sh
go test ./...   # pins every slug, status and the size-limit distinction
go run .        # shows the bodies and the slug catalog
```

No environment is required.

## Why a slug, not a number

A numeric error code needs a registry mapping numbers to meanings, and that
registry is the thing that stops being maintained. A slug - `slug_taken`,
`invalid_json`, `rate_limit` - says it in the log, the test and the frontend
`switch`, and it is what an i18n key is looked up by. `goloop/resp` carries it
beside the numeric code and the human message: `{"code":409,"error":
"slug_taken","message":"..."}`.

## The kit

- **`Fail`** - one helper for the whole service, so every error has the same
  shape.
- **`Decode`** - a JSON body reader with a size limit that answers `invalid_json`
  for a broken body and `body_too_large` for one over the cap; the frontend
  tells the two apart.
- **`qp.ParseInt(...).Error`** - a bad query value becomes a `400` with a slug,
  not a silent fallback that hides the caller's mistake.
- **the catalog** - a fixed table of slug -> status -> i18n key, the contract the
  frontend shares.
