[![Stay with Ukraine](https://img.shields.io/static/v1?label=Stay%20with&message=Ukraine%20♥&color=ffD700&labelColor=0057B8&style=flat)](https://u24.gov.ua/)

# goloop

📖 **English** · [Українська](README.UK.md)

`goloop` is a group of small, focused Go modules for the everyday work around
configuration, command-line tools, HTTP handlers, validation, logging,
collections, identifiers, strings and three-valued logic. The modules are
independent: you import only the package you need, and each package keeps its
own versioned module path.

The current group is:

`env`, `g`, `is`, `key`, `log`, `opt`, `qp`, `resp`, `scs`, `set`, `slug`,
`t13n`, `trit`.

Together they cover the boring but important edges of application code: reading
configuration from `.env` files, parsing CLI arguments, validating user input,
reading query parameters, writing HTTP responses, producing logs, converting
string styles, building slugs, transliterating Unicode text, working with sets,
short reversible keys, generic helpers and nullable/unknown boolean logic.

## Contents

Jump to a package; each section ends with links to its repository and reference.

- [**env** — .env files, process environment and struct mapping](#env)
- [**g** — generic helpers for slices, numbers, conditions and conversions](#g)
- [**is** — format and value validation](#is)
- [**key** — reversible short keys for uint64 IDs](#key)
- [**log** — multi-output leveled logging](#log)
- [**opt** — command-line argument parsing into structs](#opt)
- [**qp** — typed URL query parameter parsing](#qp)
- [**resp** — HTTP response helpers on top of net/http](#resp)
- [**scs** — string case conversion and detection](#scs)
- [**set** — generic comparable sets](#set)
- [**slug** — URL-friendly slugs from Unicode text](#slug)
- [**t13n** — Unicode-to-ASCII transliteration](#t13n)
- [**trit** — three-valued logic: False, Unknown, True](#trit)

## env

`env` connects `.env` files, the process environment and Go structs. Use it
when configuration is stored in environment variables but the application wants
a typed config object with defaults, required fields, slices, arrays, nested
structs, `time.Duration`, `time.Time`, `url.URL` and other ordinary Go types.

It can load files into the process environment (`Load`, `Overload`), parse
`.env` data into maps without side effects (`Read`, `Parse`), unmarshal the
environment into a struct, and marshal a struct back to `.env` text or files.
That makes it useful both for applications and for tests where mutating global
environment state is undesirable.

```go
package main

import (
	"log"
	"time"

	"github.com/goloop/env/v2"
)

type Config struct {
	Host    string        `env:"HOST" def:"127.0.0.1"`
	Port    int           `env:"PORT" def:"8080"`
	Timeout time.Duration `env:"TIMEOUT" def:"5s"`
}

func main() {
	var cfg Config
	if err := env.Unmarshal(&cfg); err != nil {
		log.Fatal(err)
	}
}
```

**Learn more:** [github.com/goloop/env](https://github.com/goloop/env) · [reference](https://pkg.go.dev/github.com/goloop/env/v2)

## g

`g` is a generic helper toolbox. It collects compact functions that Go
programs often rewrite locally: conditional values, lazy conditionals, min/max
helpers, clamp, mapping and filtering slices, membership checks, conversions,
safe arithmetic and common numeric utilities.

The package is best used as a small convenience layer around standard-library
semantics. In v2 the hot paths lean on modern Go packages such as `slices`,
`maps`, `cmp`, `iter` and `math/rand/v2` where that is the correct thing to do,
while keeping a short `g.*` facade for application code.

```go
package main

import (
	"fmt"

	g "github.com/goloop/g/v2"
)

func main() {
	name := g.If(len("admin") > 0, "admin", "guest")
	page := g.Clamp(250, 1, 100)
	ids := g.Filter([]int{1, 2, 3, 4}, func(v int) bool { return v%2 == 0 })

	fmt.Println(name, page, ids) // admin 100 [2 4]
}
```

**Learn more:** [github.com/goloop/g](https://github.com/goloop/g) · [reference](https://pkg.go.dev/github.com/goloop/g/v2)

## is

`is` is a validation package. Each function answers one question about a value:
is this an email, an IP address, an IBAN, a UUID, a phone number, a hex color,
a JWT, a numeric string, a valid latitude, a variable name, and so on.

The package validates the input as given; it is not a sanitizer or normalizer.
That distinction is important in HTTP handlers and forms: normalize data first
if your application needs normalization, then call `is.*` to check whether the
result matches the expected format or rule.

```go
package main

import (
	"fmt"

	"github.com/goloop/is/v2"
)

func main() {
	fmt.Println(is.Email("user@example.com"))                    // true
	fmt.Println(is.UUID("550e8400-e29b-41d4-a716-446655440000")) // true
	fmt.Println(is.IP("2001:db8::1"))                            // true
}
```

**Learn more:** [github.com/goloop/is](https://github.com/goloop/is) · [reference](https://pkg.go.dev/github.com/goloop/is/v2)

## key

`key` converts `uint64` identifiers to short reversible string keys using a
custom alphabet. It is useful for public IDs, invite codes, ticket numbers,
coupon codes and URL-safe representations of internal numeric IDs.

The core abstraction is a `Locksmith`: a base-N encoder/decoder over your
alphabet. Dynamic keys are as short as the number allows; fixed keys always
have exactly the requested size. Decoding is strict, so every valid key has one
canonical textual form and one numeric ID.

```go
package main

import (
	"fmt"

	"github.com/goloop/key/v2"
)

func main() {
	ls := key.MustNewFixed(key.Base62, 8)

	s, _ := ls.Marshal(12345)
	id, _ := ls.Unmarshal(s)

	fmt.Println(s, id) // 000003D7 12345
}
```

**Learn more:** [github.com/goloop/key](https://github.com/goloop/key) · [reference](https://pkg.go.dev/github.com/goloop/key/v2)

## log

`log` is a leveled logger with multiple outputs. It can write different levels
to different writers, render text or JSON, include timestamps and caller
layout fields, add prefixes, use colors for terminals and report write errors
through an error handler.

In modern Go it should be treated as a practical logging facade and output
router, not as a replacement for every `log/slog` use case. It is useful when
an application wants simple multi-destination logging with level masks and
format control while still staying close to Go's standard logging model.

```go
package main

import "github.com/goloop/log/v2"

func main() {
	logger := log.New("APP")

	logger.Info("service started")
	logger.Warnf("cache miss for key %q", "user:42")
	logger.Error("background job failed")
}
```

**Learn more:** [github.com/goloop/log](https://github.com/goloop/log) · [reference](https://pkg.go.dev/github.com/goloop/log/v2)

## opt

`opt` parses command-line arguments into a struct. It is for CLI programs that
want a typed configuration object instead of manual `os.Args` scanning and
`strconv` calls. Fields are configured with tags for short flags, long aliases,
defaults, help text, separators, required values and positional arguments.

The v2 parser follows normal Unix/POSIX expectations: boolean flags are
switches, `--no-name` disables a bool, `--flag=value` is accepted, `--`
terminates option parsing, and negative numbers can be values or positional
arguments where appropriate.

```go
package main

import (
	"log"

	"github.com/goloop/opt/v2"
)

type Args struct {
	Host    string `opt:"H" alt:"host" def:"127.0.0.1"`
	Port    int    `opt:"p" alt:"port" def:"8080"`
	Verbose bool   `opt:"v" alt:"verbose"`
	Files   []string `opt:"[]"`
}

func main() {
	var args Args
	if err := opt.Unmarshal(&args); err != nil {
		log.Fatal(err)
	}
}
```

**Learn more:** [github.com/goloop/opt](https://github.com/goloop/opt) · [reference](https://pkg.go.dev/github.com/goloop/opt/v2)

## qp

`qp` reads URL query parameters into typed Go values. It replaces the repeated
pattern of `r.URL.Query().Get(...)`, `strconv.Atoi`, default handling and
range checks with one compact API.

Use `qp.New(r.URL)` when a handler reads several parameters from the same URL;
it parses the query once and then exposes typed readers. Top-level helpers are
available for one-off reads. Options cover defaults, ranges, allowed values,
slice splitting and per-element validation.

```go
package main

import (
	"net/http"

	"github.com/goloop/qp/v2"
)

func handler(w http.ResponseWriter, r *http.Request) {
	q := qp.New(r.URL)

	page := q.Int("page", qp.Default(1), qp.Min(1))
	limit := q.Int("limit", qp.Default(20), qp.Between(1, 100))
	tags := q.StringSlice("tag")

	_, _, _ = page, limit, tags
}
```

**Learn more:** [github.com/goloop/qp](https://github.com/goloop/qp) · [reference](https://pkg.go.dev/github.com/goloop/qp/v2)

## resp

`resp` is a thin helper layer over `net/http` for writing common HTTP
responses. It covers JSON, JSONP, XML, HTML, strings, bytes, redirects,
downloads, cookies, status codes and headers without becoming a web framework.

The important v2 detail is safe-by-default encoding: JSON/JSONP/XML are
encoded into a pooled buffer first, so serialization errors are returned before
the HTTP status is committed. You can opt into direct streaming for large
payloads when that trade-off is better.

```go
package main

import (
	"net/http"

	"github.com/goloop/resp/v2"
)

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("id") == "" {
		_ = resp.Error(w, http.StatusBadRequest, "missing id")
		return
	}

	_ = resp.JSON(w, resp.R{"ok": true}, resp.SecurityHeaders())
}
```

**Learn more:** [github.com/goloop/resp](https://github.com/goloop/resp) · [reference](https://pkg.go.dev/github.com/goloop/resp/v2)

## scs

`scs` means String Case Style. It converts identifiers between `camelCase`,
`PascalCase`, `snake_case`, `kebab-case`, `SCREAMING_SNAKE_CASE`, `dot.case`,
`Title Case` and `Sentence case`.

All converters use one tokenizer, so you do not need to know the input style in
advance. The package can also detect a style when the answer is unambiguous,
split text into words, iterate words with `iter.Seq`, and use an immutable
`Caser` for opt-in acronyms such as `ID`, `URL` and `HTTP`.

```go
package main

import (
	"fmt"

	"github.com/goloop/scs/v2"
)

func main() {
	fmt.Println(scs.ToSnake("HTTPServerID")) // http_server_id
	fmt.Println(scs.ToCamel("user_id"))      // userId

	c := scs.New(scs.WithAcronyms("ID", "URL"))
	fmt.Println(c.ToPascal("user_id")) // UserID
}
```

**Learn more:** [github.com/goloop/scs](https://github.com/goloop/scs) · [reference](https://pkg.go.dev/github.com/goloop/scs/v2)

## set

`set` is a generic set for comparable Go values. It is built directly on
`map[T]struct{}`, so identity is exactly Go's `==`: no reflection, no custom
hashing and no collision-based loss of elements.

Use it for deduplication, membership checks, set algebra and relation checks:
union, intersection, difference, symmetric difference, subset/superset and
disjointness. It also includes functional helpers, JSON support and iteration
through `iter.Seq`.

```go
package main

import (
	"fmt"

	"github.com/goloop/set/v2"
)

func main() {
	a := set.New(1, 2, 3)
	b := set.New(3, 4)

	fmt.Println(set.Sorted(a.Union(b)))        // [1 2 3 4]
	fmt.Println(a.Contains(2), a.Contains(9)) // true false
}
```

**Learn more:** [github.com/goloop/set](https://github.com/goloop/set) · [reference](https://pkg.go.dev/github.com/goloop/set/v2)

## slug

`slug` generates URL-friendly slugs from Unicode text. It uses `t13n` for
transliteration and then normalizes the result into words joined by a
separator. Punctuation becomes a boundary instead of silently merging words.

Use the package-level helpers for simple cases, or build an immutable `Slug`
with options for language rules, a custom separator, maximum length and a
fallback for empty results. It can also validate whether a string is already a
canonical slug and generate a unique slug against your storage predicate.

```go
package main

import (
	"fmt"

	"github.com/goloop/slug/v2"
	"github.com/goloop/slug/v2/lang"
)

func main() {
	s := slug.New(slug.WithLang(lang.UK), slug.WithMaxLength(32))

	fmt.Println(slug.Lower("Hello, World!")) // hello-world
	fmt.Println(s.Make("Привіт, світ!"))    // Pryvit-svit
}
```

**Learn more:** [github.com/goloop/slug](https://github.com/goloop/slug) · [reference](https://pkg.go.dev/github.com/goloop/slug/v2)

## t13n

`t13n` means transliteration. It converts Unicode text to ASCII, optionally
applying regional language rules. It is the lower-level text conversion engine
behind `slug`, but it is also useful on its own when you need ASCII-only search
keys, filenames, identifiers or logs.

The package exposes single-rune conversion, whole-string conversion,
language-specific conversion and custom rendering rules. The base table is
embedded compactly and decoded lazily, so applications pay for it when they
actually transliterate.

```go
package main

import (
	"fmt"

	"github.com/goloop/t13n/v2"
	"github.com/goloop/t13n/v2/lang"
)

func main() {
	fmt.Println(t13n.Make("世界"))                         // Shi Jie
	fmt.Println(t13n.Trans(lang.UK, "Доброго вечора")) // Dobroho vechora

	s, ok := t13n.Rune('界')
	fmt.Println(s, ok) // Jie  true
}
```

**Learn more:** [github.com/goloop/t13n](https://github.com/goloop/t13n) · [reference](https://pkg.go.dev/github.com/goloop/t13n/v2)

## trit

`trit` implements three-valued logic: `False`, `Unknown` and `True`. It is
useful when a value is not simply yes/no: nullable database booleans, partially
known configuration, feature flags with inherited defaults, policy decisions
or any domain where “unknown” must not be collapsed into `false`.

The zero value is `Unknown`, which makes uninitialized values meaningful.
The package provides truth-table operations, generic aggregate functions,
parsing, JSON/text/SQL integration and ordering (`False < Unknown < True`).

```go
package main

import (
	"fmt"

	"github.com/goloop/trit/v2"
)

func main() {
	enabled := trit.Unknown
	enabled.Default(trit.True)

	fmt.Println(enabled.And(trit.True))              // True
	fmt.Println(trit.Consensus(trit.True, enabled)) // True
}
```

**Learn more:** [github.com/goloop/trit](https://github.com/goloop/trit) · [reference](https://pkg.go.dev/github.com/goloop/trit/v2)

## How to choose

Use `env` and `opt` at program startup, `qp` and `resp` in HTTP handlers,
`is` for validation, `log` for operational output, `set` and `g` inside
business logic, `key` for public reversible IDs, `scs`, `slug` and `t13n` for
string processing, and `trit` whenever unknown state is a first-class value.

Each module is intentionally small. You do not need to adopt the whole group:
install only the module that closes the specific problem in front of you.
