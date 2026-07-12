[« Sessions, tokens and passwords](06-auth.md) · [Contents](../main.md) · [A service lifecycle »](08-lifecycle.md)

---

# 07. Slugs, transliteration and cases

**Task.** Turn a human title into a clean URL slug, even when the title is in
Cyrillic; keep the slug unique; and convert an identifier between naming styles.

**Modules.** [`slug`](https://github.com/goloop/slug) makes URL slugs,
[`t13n`](https://github.com/goloop/t13n) transliterates non-Latin text to
ASCII, and [`scs`](https://github.com/goloop/scs) converts between snake, kebab,
camel, Pascal and more.

**Recipe.** [`recipes/007-slug`](../recipes/007-slug/)

## Example A - a unique slug

`slug.New(slug.WithLowercase())` gives a maker that returns lower-case slugs;
`MakeUnique` appends `-2`, `-3`, ... while a callback reports a clash:

```go
s := slug.New(slug.WithLowercase())
taken := map[string]bool{"getting-started": true}
out := s.MakeUnique("Getting Started", func(x string) bool { return taken[x] })
// out == "getting-started-2"
```

## Example B - transliterate, then slug

A Cyrillic title has no ASCII slug on its own. `t13n` transliterates it to
Latin first; `slug` then makes the URL segment:

```go
tr := t13n.New()
latin := tr.Make("Привіт, світ") // "Privit, svit"
url := s.Make(latin)             // "privit-svit"
```

## Example C - naming styles

One `scs.Caser` converts a name between every common style:

```go
c := scs.New()
c.ToSnake("userAPIToken")          // "user_api_token"
c.ToKebab("userAPIToken")          // "user-api-token"
c.ToPascal("userAPIToken")         // "UserApiToken"
c.ToScreamingSnake("userAPIToken") // "USER_API_TOKEN"
```

## Example D - validate an incoming slug

A slug does not only go out in a URL you build; one also comes back in, as the
`{slug}` in a request path. `slug.IsValid` answers yes/no without changing the
value, so a handler can reject a malformed segment before it touches the
database:

```go
slug.IsValid("hello-world")  // true
slug.IsValid("Hello World!") // false - spaces and punctuation
slug.IsValid("-bad-")        // false - leading/trailing dash
```

## Example E - keep acronyms upper-case

By default `scs` does not know `API` is an acronym, so `userApiToken` becomes
`UserApiToken`. Teach it the acronyms you use with `WithAcronyms`, and they stay
upper-case in every style:

```go
acr := scs.New(scs.WithAcronyms("API", "ID"))
acr.ToPascal("userApiToken") // "UserAPIToken" (plain scs: "UserApiToken")
acr.ToPascal("user_id")      // "UserID"       (plain scs: "UserId")
```

## Execution report

```text
$ go test ./...
ok  	goloop.one/handbook/007-slug	0.005s

$ go run .
A. unique URL slug (slug):
   "Getting Started"  -> "getting-started-2"
   "Getting Started"  -> "getting-started-3"
   "Hello, World!"    -> "hello-world"
B. transliterate then slug (t13n + slug):
   "Привіт, світ"       -> "Privit, svit"         -> "privit-svit"
   "Огляд архітектури"  -> "Ogliad arkhitekturi"  -> "ogliad-arkhitekturi"
C. naming styles (scs):
   from "userAPIToken":
     snake="user_api_token" kebab="user-api-token"
     pascal="UserApiToken" camel="userApiToken"
     screaming="USER_API_TOKEN" title="User Api Token"
D. validate an incoming slug (slug.IsValid):
   "hello-world"    -> IsValid=true
   "Hello World!"   -> IsValid=false
   "-bad-"          -> IsValid=false
E. acronyms in case conversion (scs WithAcronyms):
   "userApiToken" -> plain pascal="UserApiToken", acronym pascal="UserAPIToken"
   "user_id"      -> plain pascal="UserId", acronym pascal="UserID"
```

The two "Getting Started" titles became `-2` and `-3` because the first was
already taken; the Cyrillic titles transliterated and then slugged; and one
`Caser` produced every naming style from the same input. In D `IsValid` accepted
the well-formed slug and rejected the two malformed ones; in E `WithAcronyms`
kept `API` and `ID` upper-case where plain `scs` title-cased them.

## What you learned

- `slug.New(slug.WithLowercase())` makes lower-case URL slugs; `MakeUnique`
  keeps them unique against your own "is it taken" check.
- `slug.IsValid` validates a slug that arrives in a request path, so a handler
  can reject a malformed segment early.
- `t13n` transliterates non-Latin text to ASCII, so a Cyrillic or other title
  gets a real slug when you run it through `t13n` before `slug`.
- `scs.Caser` converts one identifier between snake, kebab, camel, Pascal,
  screaming-snake and title case; `scs.WithAcronyms` keeps known acronyms
  upper-case.

Part III begins: running these pieces as a service.

---

[« Sessions, tokens and passwords](06-auth.md) · [Contents](../main.md) · [A service lifecycle »](08-lifecycle.md)
