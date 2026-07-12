[« JSON HTTP API](02-http-json-api.md) · [Contents](../main.md) · [Contents »](../main.md)

---

# 03. Validate and clean user input

**Task.** A signup form arrives full of human mess: stray spaces, mixed case, a
zero-width character pasted from a chat app, a phone number written with dashes
and parentheses. Reject what is truly invalid, and quietly canonicalize what is
just untidy - before any of it reaches your database.

**Modules.** [`is`](https://github.com/goloop/is) answers *"is this valid?"*
with read-only predicates; [`norm`](https://github.com/goloop/norm) answers
*"make this the canonical form"* by cleaning and folding.

**Recipe.** [`recipes/003-validate-and-clean`](../recipes/003-validate-and-clean/)

## The idea

Input handling is two jobs, and mixing them causes bugs. Some fields must be
*exactly* valid (an email either is or is not an address). Other fields should
be *forgiving* (a name with a trailing space is fine, just trim it). `is` is the
read-only half - it never changes the value, it only reports. `norm` is the
write half - it returns a cleaned value and, where relevant, whether it is
valid.

Cleaning a free-text field is one call:

```go
name := norm.Clean(f.Name) // strip invisible/control bytes, collapse spaces, trim
if name == "" {
	return Account{}, ErrEmptyName
}
```

`norm.Clean` removes invisible and control characters (including that pasted
zero-width space), collapses every run of whitespace to a single space, and
trims the ends. A field that cleans to nothing was empty.

An email is an identity, so fold its case:

```go
email, ok := norm.EmailFold(f.Email) // validate + lower-case the whole address
if !ok {
	return Account{}, ErrInvalidEmail
}
```

`norm.EmailFold` validates and lower-cases the whole address, so
`Ada@Example.COM` and `ada@example.com` are the same account. (`norm.Email`
exists too, and preserves the local part's case for exact deliverability - use
`EmailFold` when the address is a lookup key.)

An optional field is cleaned when present and dropped when not:

```go
phone := ""
if f.Phone != "" {
	if e164, ok := norm.E164(f.Phone); ok {
		phone = e164 // +14155550132
	}
}
```

## Execution report

Four forms through `process`, then the read-only `is` predicates directly:

```text
$ go test ./...
ok  	goloop.one/handbook/003-validate-and-clean	0.004s

$ go run .
form 1: name="Ada Lovelace" email="ada@example.com" phone="+14155550132"
form 2: name="Grace Hopper" email="grace@example.com" phone=""
form 3: rejected: a name is required
form 4: rejected: a valid email is required
---
is.Email("a@b.co") = true
is.Numeric("12345") = true
```

Form 1 came in as `"  Ada​ Lovelace "` (with a zero-width space after "Ada") and
`"  Ada@Example.COM "` and `"+1 (415) 555-0132"`; it left clean, folded and in
E.164. Form 3's name was a lone tab, which cleans to nothing, so it is rejected.
Form 4's email is not an address, so it is rejected. Nothing untrusted slipped
through, and nothing merely untidy was thrown away.

## What you learned

- Split the job: **validate** what must be exact (`is`), **clean** what should
  be forgiving (`norm`).
- `norm.Clean` is the safe one-call cleanup for any free-text field.
- `norm.EmailFold` gives a case-insensitive identity key; `norm.Email` preserves
  case for deliverability.
- Optional fields (`norm.E164` and friends) return `(value, ok)`, so you can
  clean-if-present without rejecting the whole form.

You have reached the end of Part I: configuration in, requests routed, input
cleaned. Part II moves on to storing that clean data in PostgreSQL and asking a
language model about it.

---

[« JSON HTTP API](02-http-json-api.md) · [Contents](../main.md) · [Contents »](../main.md)
