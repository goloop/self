[« AI, production shape](14-ai-production.md) · [Contents](../main.md)

---

# 15. Errors a frontend can switch on

**Task.** When a request fails, a JSON API owes the client two things: a message
a human can read, and a code the program can act on. The number most APIs send
is a poor code - `1042` says nothing in a log, a test or a `switch`, and turning
it into a meaning needs a registry that nobody keeps up to date. A slug says it
plainly - `slug_taken`, `invalid_json`, `rate_limit` - and the frontend switches
on it and looks up the translated message by it, in every language, without
trusting the prose the backend happened to send.

**Modules.** [`resp`](https://github.com/goloop/resp) carries the slug beside the
numeric code and the message; [`qp`](https://github.com/goloop/qp) reads and
bounds query parameters; [`mux`](https://github.com/goloop/mux) routes.

**Recipe.** [`recipes/015-error-slugs`](../recipes/015-error-slugs/)

## The shape

Every error the service sends has the same three fields, from one helper, so no
handler invents its own:

```go
func Fail(w http.ResponseWriter, status int, slug, message string) {
	_ = resp.Error(w, status, message, resp.WithErrorSlug(slug))
}

// Fail(w, 409, "slug_taken", "That slug is already in use") writes:
//   HTTP 409  {"code":409,"error":"slug_taken","message":"That slug is already in use"}
```

`code` is the HTTP status mirrored into the body; `error` is the machine slug;
`message` is for a human. The frontend reads `error`, ignores `message` except to
show it as a fallback, and looks up its own translation by the slug.

## Example A - a decode kit that names the failure

Reading a JSON body has two distinct failure modes, and telling them apart
matters: a broken body sends a developer to their serializer, which is the wrong
place when the body was simply cut off at a size limit. `Decode` answers the
right slug for each:

```go
func Decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBytes) // the cap
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			Fail(w, 413, "body_too_large", "Request body exceeds the maximum size")
			return false
		}
		Fail(w, 400, "invalid_json", "Request body is not valid JSON")
		return false
	}
	return true
}
```

The size limit lives on `Decode` - the one function every JSON handler calls -
so it covers routes that do not exist yet, without a hand-maintained list of
"JSON only" routes whose failure mode (forgetting to add a new route) is silent.
This is the pattern a real service used instead of `middlewares.MaxBytes` at the
router level, because its upload routes needed a ceiling thousands of times
larger and set their own.

## Example B - a bad query value is an error, not a default

`qp.Int` returns the default on a bad value without complaint, which is right for
an optional knob. Where a bad value should be an error - pagination typed wrong -
`ParseInt` returns the full `Result`, and its `Error` field carries the reason,
so a mistake gets a slug instead of a silent reset to page 1:

```go
q := qp.New(r.URL)
pageR := q.ParseInt("page", qp.Default(1), qp.Min(1))
perPageR := q.ParseInt("per_page", qp.Default(20), qp.Min(1), qp.Max(100))
if pageR.Error != nil || perPageR.Error != nil {
	Fail(w, 400, "invalid_query", "page and per_page must be positive integers")
	return
}
```

The bounds (`Min`, `Max`) do double duty: they validate, and they stop a caller
from asking for a million rows.

## Example C - the catalog

The set of slugs a service answers is a contract the frontend shares. Keep it in
one table - generated, or shared with the frontend, or at least written down:

| Slug | Status | i18n key |
|---|---|---|
| `invalid_json` | 400 | `errors.invalid_json` |
| `invalid_query` | 400 | `errors.invalid_query` |
| `slug_required` | 422 | `errors.slug_required` |
| `slug_taken` | 409 | `errors.slug_taken` |
| `body_too_large` | 413 | `errors.body_too_large` |

Slugs are the application's own vocabulary, so they are not validated by `resp` -
keep them to a fixed set shaped like `[a-z0-9_]+`. The value is emitted verbatim,
which makes it the wrong place for anything a request supplied.

## Execution report

```
$ go run .
A/B. one shape, right slug per failure:
   bad JSON:        400 {"code":400,"error":"invalid_json","message":"Request body is not valid JSON"}
   missing slug:    422 {"code":422,"error":"slug_required","message":"A slug is required"}
   slug taken:      409 {"code":409,"error":"slug_taken","message":"That slug is already in use"}
   created:         200 {"slug":"fresh"}
C. query params, a slug on a bad value:
   good:            200 {"page":2,"per_page":50}
   bad page:        400 {"code":400,"error":"invalid_query","message":"page and per_page must be positive integers"}
D. the catalog the frontend switches on:
   invalid_json   -> 400  errors.invalid_json
   ...
```

## What you learned

- A slug (`resp.WithErrorSlug`) is what a client switches on and an i18n key is
  looked up by; a number needs a registry that rots.
- One `Fail` helper gives every error the same `{code, error, message}` shape.
- A decode kit with a size limit answers `invalid_json` and `body_too_large`
  distinctly, and lands the limit where every JSON handler already calls.
- `qp.ParseInt(...).Error` turns a bad query value into a slugged 400, not a
  silent default.
- Keep the slugs in one catalog - it is the contract the frontend shares.

---

[« AI, production shape](14-ai-production.md) · [Contents](../main.md) · [A catalog people edit by hand »](16-yaml-catalog.md)
