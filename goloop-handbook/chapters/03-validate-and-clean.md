[« JSON HTTP API](02-http-json-api.md) · [Contents](../main.md) · [Contents »](../main.md)

---

# 03. Validate and clean user input

**Task.** A signup form arrives full of human mess: stray spaces, mixed case, a
zero-width character pasted from a chat app, a phone number written with dashes
and parentheses. Reject what is truly invalid, and quietly canonicalize what is
just untidy - before any of it reaches your database.

**Modules.** [`is`](https://github.com/goloop/is) answers *"is this valid?"* with
read-only predicates; [`norm`](https://github.com/goloop/norm) answers *"make
this the canonical form"* by cleaning and folding.

**Recipe.** [`recipes/003-validate-and-clean`](../recipes/003-validate-and-clean/)

## The two halves

Input handling is two jobs, and mixing them causes bugs. Some fields must be
*exactly* valid; others should be *forgiving*. `is` is the read-only half - it
never changes the value, it only reports. `norm` is the write half - it returns
a cleaned value and, where relevant, whether it is valid.

## Example A - the signup form

```go
name := norm.Clean(f.Name) // strip invisible/control bytes, collapse spaces, trim
if name == "" {
	return Account{}, ErrEmptyName
}
email, ok := norm.EmailFold(f.Email) // validate + lower-case the whole address
if !ok {
	return Account{}, ErrInvalidEmail
}
phone := ""
if f.Phone != "" {
	if e164, ok := norm.E164(f.Phone); ok {
		phone = e164 // +14155550132
	}
}
```

`norm.Clean` removes invisible and control characters (including a pasted
zero-width space), collapses whitespace and trims. `norm.EmailFold` validates
and lower-cases the whole address so `Ada@Example.COM` and `ada@example.com` are
the same account. An optional field is cleaned when present and dropped when not.

## Example B - the `is` gallery

When you only need a yes/no - on a query parameter or a config value - use `is`
directly. It never mutates the input:

```go
is.Email("a@b.co")   // true
is.URL("https://goloop.one") // true
is.UUID("9b1deb4d-...")      // true
is.Numeric("12345")  // true
is.Numeric("12a3")   // false - letters are not numeric
```

## Example C - the `norm` toolkit

Beyond the typed normalizers, `norm` has a character toolkit built on classes
(`Letters`, `Digits`, `Punct`, `Invisible`, ...) to shape any value:

```go
norm.DigitsOnly("+1 (415) 555-0132") // "14155550132"
norm.AlnumOnly("user.name_42!")      // "username42"
norm.Remove("a,b;c.d", norm.Punct)   // "abcd"
norm.Keep("Go1.24!", norm.Letters)   // "Go"
```

## Execution report

```text
$ go test ./...
ok  	goloop.one/handbook/003-validate-and-clean	0.004s

$ go run .
A. clean and validate a signup form:
   form 1: name="Ada Lovelace" email="ada@example.com" phone="+14155550132"
   form 2: name="Grace Hopper" email="grace@example.com" phone=""
   form 3: rejected: a name is required
   form 4: rejected: a valid email is required
B. is predicates (read-only, never change the value):
   is.Email(a@b.co)               = true
   is.URL(https://goloop.one)     = true
   is.UUID(9b1deb...)             = true
   is.Numeric(12345)              = true
   is.Numeric(12a3)               = false
C. norm toolkit (shape a value to a canonical form):
   DigitsOnly("+1 (415) 555-0132")  = "14155550132"
   AlnumOnly("user.name_42!")   = "username42"
   Remove("a,b;c.d", Punct) = "abcd"
   Keep("Go1.24!", Letters) = "Go"
```

Form 1 came in with a zero-width space in the name, a mixed-case email with
spaces, and a formatted phone; it left clean, folded and in E.164. Form 3's
name was a lone tab, which cleans to nothing, so it is rejected. Nothing
untrusted slipped through, and nothing merely untidy was thrown away.

## What you learned

- Split the job: **validate** what must be exact (`is`), **clean** what should
  be forgiving (`norm`).
- `norm.Clean` is the safe one-call cleanup; `norm.EmailFold` gives a
  case-insensitive identity key; `norm.E164` and friends return `(value, ok)`.
- `is` predicates (`Email`, `URL`, `UUID`, `Numeric`, ...) are read-only yes/no
  answers for a parameter or config value.
- The `norm` toolkit (`DigitsOnly`, `AlnumOnly`, `Keep`, `Remove`) shapes a
  value using character classes.

You have reached the end of Part I. Part II moves on to storing this clean data
in PostgreSQL and asking a language model about it.

---

[« JSON HTTP API](02-http-json-api.md) · [Contents](../main.md) · [Contents »](../main.md)
