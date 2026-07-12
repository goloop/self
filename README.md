[![Stay with Ukraine](https://img.shields.io/static/v1?label=Stay%20with&message=Ukraine%20♥&color=ffD700&labelColor=0057B8&style=flat)](https://u24.gov.ua/)

# goloop

📖 **English** · [Українська](README.UK.md)

`goloop` is a group of small, focused Go modules for the everyday work around
configuration, command-line tools, HTTP handlers, routing and middleware,
WebSocket connections, large language model APIs, typed PostgreSQL queries,
validation, logging, collections, identifiers, strings, type reflection and
three-valued logic - plus a thin application layer (service lifecycle, health
endpoints, authentication, sessions and email) that builds on those pieces. The
modules are independent: you import only the package you need, and each package
keeps its own versioned module path.

The group has two layers. A **foundation layer** of independent, mostly
zero-dependency utilities, and a small **application layer** that composes them
into the recurring parts of a network service. New to goloop? Read
[Building a service](#building-a-service) first for the whole picture, then dip
into the per-module reference below.

The current group is:

- **Foundation:** `ai` (with the provider drivers `anthropic`, `openai`,
  `gemini`, `grok`, `deepseek`, `openrouter`, `ollama`, `mistral`, `cohere`),
  `env`, `g`, `is`, `key`, `kind`, `log`, `middlewares`, `mux`, `opt`, `pgc`,
  `qp`, `resp`, `scs`, `set`, `slug`, `t13n`, `trit`, `websocket`.
- **Application:** `app`, `observe`, `jwt`, `auth`, `argon2id`, `session`, `mail`, `cli`.

Together they cover the boring but important edges of application code: reading
configuration from `.env` files, parsing CLI arguments, validating user input,
reading query parameters, routing requests, chaining middleware, writing HTTP
responses, speaking the WebSocket protocol, producing logs, converting string
styles, building slugs, transliterating Unicode text, working with sets, short
reversible keys, generic helpers and nullable/unknown boolean logic - and then
running a service with an ordered start/stop lifecycle, health and readiness
probes, signed tokens and cookie sessions, and outbound email.

## What goloop means

GoLoop is a stdlib-first Go toolkit. It does not try to replace Go's standard
library or hide it behind a framework. Each module keeps the standard library
as the default vocabulary, then adds the small missing layer that application
code would otherwise rewrite in every project.

The design rules are practical:

- **Small modules, clear ownership.** A package should solve one ordinary
  problem well: environment config, options, responses, routing, validation,
  logging, WebSocket, generated PostgreSQL queries, and so on.
- **Explicit behavior.** Public APIs should make control flow, errors and data
  movement visible. Generated code should look like code a careful Go developer
  would write by hand.
- **Stdlib first.** Prefer `net/http`, `database/sql`, `log/slog`, `context`,
  `encoding/json`, `iter`, `slices`, `maps`, `cmp` and other standard packages
  over custom runtime abstractions.
- **Zero dependencies by default.** A module may depend on another GoLoop
  module when that is the point of the design, but third-party dependencies are
  exceptional and must pay for themselves.
- **No framework lock-in.** Use one module without adopting the rest. Generated
  code should not require a GoLoop runtime package unless that dependency is the
  feature being requested.
- **Measured quality.** Keep hot paths allocation-aware, race-clean and covered
  by tests. Benchmarks are private engineering tools, not marketing.
- **Boring surface, strong internals.** The best GoLoop package feels obvious
  from the outside, even when it contains protocol work, parsing, caching or
  code generation inside.

In short: GoLoop is for applications that want explicit Go, standard-library
interfaces, small focused packages and production-grade edges without adopting a
large framework.

This philosophy is why `pgc` generates plain `database/sql` code instead of a
runtime ORM, why `websocket` implements RFC 6455 without pulling a framework,
why `mux` builds on `net/http.ServeMux`, and why provider packages such as
`anthropic` or `openai` expose native APIs while sharing only the small `ai`
contract where it is useful.

## Building a service

This is the part to read first. The modules are independent, but they are
designed to slot together in a predictable order. A typical goloop service is
assembled in layers, each one closing a specific gap the standard library
leaves open:

| Stage | Modules | What they do |
|-------|---------|--------------|
| **Configuration** | `env`, `opt` | Read `.env`/environment into a typed config struct; parse CLI flags. |
| **Lifecycle** | `app` | Own the ordered start/stop sequence and graceful shutdown so `main` does not. |
| **HTTP edge** | `mux`, `middlewares` | Route requests over `net/http.ServeMux`; add request IDs, real IP, recovery, logging, CORS. |
| **Handlers** | `qp`, `resp`, `is` | Read typed query parameters, validate input, write JSON/other responses. |
| **Identity** | `auth`, `argon2id`, `jwt`, `session` | Hash passwords, issue and verify tokens, protect routes, keep browser sessions. |
| **Data** | `pgc` | Compile SQL into type-safe Go against PostgreSQL. |
| **Realtime / AI** | `websocket`, `ai` | Speak RFC 6455; call LLM providers behind one interface. |
| **Observability** | `observe`, `log` | Expose `/healthz` and `/readyz`; produce operational logs. |
| **Scaffolding** | `cli` | A command-line tool that generates and inspects goloop-first projects. |

The dependency direction is one-way: the application layer imports the
foundation, never the reverse. `app` knows nothing about HTTP or health; `mux`
knows nothing about the lifecycle; `observe` never imports `app`. You bridge
them in your own wiring code, which keeps every module independently usable and
independently testable.

A minimal but complete service wires configuration, lifecycle, routing,
middleware, health and responses together:

```go
package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/goloop/app"
	"github.com/goloop/env/v2"
	"github.com/goloop/middlewares"
	"github.com/goloop/mux"
	"github.com/goloop/observe"
	"github.com/goloop/resp/v2"
)

type Config struct {
	Addr string `env:"ADDR" def:":8080"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// 1. Configuration from the environment.
	var cfg Config
	if err := env.Unmarshal(&cfg); err != nil {
		return err
	}

	// 2. Health and readiness probes (observe never imports the lifecycle).
	obs := observe.New(observe.WithService("api"))
	obs.Check("self", func(context.Context) error { return nil })

	// 3. Routes over net/http.ServeMux, then a middleware chain.
	r := mux.New()
	r.Get("/healthz", obs.HealthHandler().ServeHTTP)
	r.Get("/readyz", obs.ReadyHandler().ServeHTTP)
	r.Get("/hello", func(w http.ResponseWriter, _ *http.Request) {
		_ = resp.JSON(w, resp.R{"message": "hello"})
	})
	handler := middlewares.Chain(
		middlewares.RequestID(),
		middlewares.RealIP(),
		middlewares.Recoverer(),
		middlewares.Logger(),
	)(r)

	// 4. Lifecycle: ordered start, signal-aware wait, graceful shutdown.
	a := app.New("api",
		app.WithLogger(slog.New(slog.NewTextHandler(os.Stdout, nil))),
		app.WithShutdownTimeout(10*time.Second),
	)
	a.Use(app.HTTPServer("http", &http.Server{Addr: cfg.Addr, Handler: handler}))
	return a.Run(context.Background())
}
```

Add identity by issuing tokens with `auth` (which signs HS256 JWTs through
`jwt`) and guarding routes with its bearer middleware, or keep browser state in
a signed cookie with `session`; send transactional email with `mail`; reach a
database with `pgc`. Each is a separate import you add only when you need it -
nothing above requires the rest of the group.

## Contents

Jump to a package; each section ends with links to its repository and reference.
See [Building a service](#building-a-service) for how they fit together.

**Foundation**

- [**ai** - one interface for LLM APIs, with drivers for the major providers](#ai)
- [**env** - .env files, process environment and struct mapping](#env)
- [**g** - generic helpers for slices, numbers, conditions and conversions](#g)
- [**is** - format and value validation](#is)
- [**key** - reversible short keys for uint64 IDs](#key)
- [**kind** - cached reflection for parser and decoder authors](#kind)
- [**log** - multi-output leveled logging](#log)
- [**middlewares** - net/http middleware: request ID, real IP, recovery, logging and more](#middlewares)
- [**mux** - ergonomic routing over net/http.ServeMux](#mux)
- [**norm** - normalize user input to canonical form and validate](#norm)
- [**opt** - command-line argument parsing into structs](#opt)
- [**pgc** - SQL queries compiled into type-safe Go for PostgreSQL](#pgc)
- [**qp** - typed URL query parameter parsing](#qp)
- [**resp** - HTTP response helpers on top of net/http](#resp)
- [**scs** - string case conversion and detection](#scs)
- [**set** - generic comparable sets](#set)
- [**slug** - URL-friendly slugs from Unicode text](#slug)
- [**t13n** - Unicode-to-ASCII transliteration](#t13n)
- [**trit** - three-valued logic: False, Unknown, True](#trit)
- [**websocket** - RFC 6455 WebSocket client and server](#websocket)

**Application**

- [**app** - service lifecycle: ordered start/stop and graceful shutdown](#app)
- [**observe** - health, readiness and build-info endpoints](#observe)
- [**jwt** - compact HS256 JSON Web Tokens](#jwt)
- [**auth** - password hashing, access tokens and bearer middleware](#auth)
- [**argon2id** - memory-hard Argon2id password hashing](#argon2id)
- [**session** - signed-cookie sessions for browser apps](#session)
- [**mail** - build and send outbound email over SMTP](#mail)
- [**cli** - scaffold and inspect goloop-first projects](#cli)

## ai

`ai` is one provider-agnostic interface for large language model APIs, plus the
shared request and response types every provider driver speaks. Like the
standard library's `database/sql` with its drivers, `ai` holds the common
contract - `Generate` and streaming `Stream`, messages, tools and multimodal
parts - while a separate package per provider implements it. Code written
against the interface runs on any provider; endpoints a provider does not share
are exposed as that driver's own native methods. Every driver depends only on
`ai`, so the whole set stays free of third-party dependencies.

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/goloop/ai"
	"github.com/goloop/anthropic"
)

func main() {
	var client ai.Client = anthropic.New(os.Getenv("ANTHROPIC_API_KEY"))

	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:    anthropic.ModelClaudeSonnet5,
		Messages: []ai.Message{ai.UserText("Say hello in one word.")},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Text())
}
```

The provider drivers, each implementing `ai.Client` and importing only `ai`:

- **anthropic** - Claude Messages API, batches and token counting - [repo](https://github.com/goloop/anthropic) · [reference](https://pkg.go.dev/github.com/goloop/anthropic)
- **openai** - Chat Completions, the Responses API, embeddings, images and audio - [repo](https://github.com/goloop/openai) · [reference](https://pkg.go.dev/github.com/goloop/openai)
- **gemini** - Google Gemini `generateContent`, embeddings and token counting - [repo](https://github.com/goloop/gemini) · [reference](https://pkg.go.dev/github.com/goloop/gemini)
- **grok** - xAI Grok, chat-completions compatible, with image generation - [repo](https://github.com/goloop/grok) · [reference](https://pkg.go.dev/github.com/goloop/grok)
- **deepseek** - DeepSeek chat and the reasoning model's chain-of-thought - [repo](https://github.com/goloop/deepseek) · [reference](https://pkg.go.dev/github.com/goloop/deepseek)
- **openrouter** - the OpenRouter gateway to many models behind one key - [repo](https://github.com/goloop/openrouter) · [reference](https://pkg.go.dev/github.com/goloop/openrouter)
- **ollama** - local models over the native Ollama API - [repo](https://github.com/goloop/ollama) · [reference](https://pkg.go.dev/github.com/goloop/ollama)
- **mistral** - Mistral chat, embeddings and fill-in-the-middle - [repo](https://github.com/goloop/mistral) · [reference](https://pkg.go.dev/github.com/goloop/mistral)
- **cohere** - Cohere v2 chat, embeddings and rerank - [repo](https://github.com/goloop/cohere) · [reference](https://pkg.go.dev/github.com/goloop/cohere)

**Learn more:** [github.com/goloop/ai](https://github.com/goloop/ai) · [reference](https://pkg.go.dev/github.com/goloop/ai)

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
if your application needs normalization (see [norm](#norm)), then call `is.*` to
check whether the result matches the expected format or rule.

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

## kind

`kind` answers one question - "what is this type, and what can it do?" - for
code that must handle arbitrary Go types at runtime. Its home ground is
parsers, decoders and binders: env vars, config files, CLI flags, query
parameters or database rows being pushed into structs the *caller* defines.
Instead of hand-rolling `reflect` ("is this an int? a pointer to a struct? does
it implement `TextUnmarshaler` on a pointer receiver?"), you ask one cached
descriptor with a flat vocabulary of predicates. Beyond parsing it covers
capability detection (does the type - or a pointer to it - implement
`sql.Scanner`, `flag.Value`, a `Set(string) error` method?) and struct/tag
walking for validators and generators; descriptors are cached per type, so hot
parse loops pay for classification once. It is deliberately narrow: if your
types are known at compile time, if one small `reflect` check would do, or if
you need to *write* values rather than classify them, you do not need `kind`.

```go
package main

import (
	"fmt"

	"github.com/goloop/kind"
)

func main() {
	k := kind.Of([]int{1, 2, 3})

	fmt.Println(k.IsSlice())      // true
	fmt.Println(k.Elem().IsInt()) // true - the element type
	fmt.Println(k.IsAnyInt())     // true - leaf-aware: the slice's leaf is int
}
```

**Learn more:** [github.com/goloop/kind](https://github.com/goloop/kind) · [reference](https://pkg.go.dev/github.com/goloop/kind)

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

## middlewares

`middlewares` is a set of HTTP middleware for the standard `net/http`. Every
middleware has the ordinary `func(http.Handler) http.Handler` shape, so it works
with any router: the standard `http.ServeMux`, the `mux` router or hand-written
handlers.

It is not a framework. It closes the common cross-cutting needs the standard
library leaves out - request identifiers, real client IP, panic recovery,
request logging, timeouts, response compression, concurrency throttling, CORS
and security headers - and logs through the standard `log/slog`.

```go
package main

import (
	"net/http"

	"github.com/goloop/middlewares"
)

func main() {
	mux := http.NewServeMux()
	// ... register handlers on mux ...

	h := middlewares.Chain(
		middlewares.RequestID(),
		middlewares.RealIP(),
		middlewares.Recoverer(),
		middlewares.Logger(),
		middlewares.Compress(),
	)(mux)

	http.ListenAndServe(":8080", h)
}
```

**Learn more:** [github.com/goloop/middlewares](https://github.com/goloop/middlewares) · [reference](https://pkg.go.dev/github.com/goloop/middlewares)

## mux

`mux` is a small routing layer over the standard `net/http.ServeMux`. Since Go
1.22 the standard multiplexer already understands method patterns, wildcard
segments and precedence, so `mux` does not replace it: it adds the ergonomics
the standard library leaves out - method helpers, prefix groups, middleware
chains and an optional error-returning handler.

The patterns are plain `net/http.ServeMux` patterns, not a custom syntax, and a
`Router` is itself an `http.Handler`, so it composes with the rest of
`net/http`.

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/goloop/mux"
)

func main() {
	r := mux.New()

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("ok"))
	})

	r.Route("/api/v1", func(r *mux.Router) {
		r.Get("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprintf(w, "user %s", req.PathValue("id"))
		})
	})

	http.ListenAndServe(":8080", r)
}
```

**Learn more:** [github.com/goloop/mux](https://github.com/goloop/mux) · [reference](https://pkg.go.dev/github.com/goloop/mux)

## norm

`norm` is the writing companion to `is`: where `is` reads (is this an email?),
`norm` cleans a value toward the canonical form of that type and then validates
it. Each function returns the normalized value and whether it is valid, so a
form handler can accept slightly messy input and store a clean result. It covers
email, URL, UUID, MAC, IBAN, bank card, phone, IP and more, and adds a Unicode
character toolkit for trimming and stripping invisible or control characters.

```go
package main

import (
	"fmt"

	"github.com/goloop/norm"
)

func main() {
	fmt.Println(norm.Email("  Example @ Gmail.com ")) // Example@gmail.com true

	iban, ok := norm.IBAN("de89 3704 0044 0532 0130 00")
	fmt.Println(iban, ok) // DE89370400440532013000 true

	// Clean strips invisible/control characters and collapses whitespace.
	fmt.Printf("%q\n", norm.Clean("  ragged   name  ")) // "ragged name"
}
```

**Learn more:** [github.com/goloop/norm](https://github.com/goloop/norm) · [reference](https://pkg.go.dev/github.com/goloop/norm)

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

## pgc

`pgc` compiles annotated SQL queries into a type-safe Go package for
PostgreSQL. You write plain SQL next to your migrations; pgc asks your
development database what every parameter and column really is - the server
itself is the type oracle, statements are prepared and described but never
executed - and generates code that reads like a person wrote it: explicit
`Scan` calls, godoc on every symbol, no reflection at runtime. The generated
package imports only the standard library (`database/sql`, `context`,
`time`), and pgc itself has zero third-party dependencies: it speaks the
PostgreSQL wire protocol directly.

```sql
-- queries/users.sql

-- name: GetUser :one
-- Returns a single user by primary key.
SELECT id, email, name, created_at
FROM users
WHERE id = $1;
```

```go
q := db.New(sqlDB) // the generated package; *sql.DB or *sql.Tx

u, err := q.GetUser(ctx, 42)   // (User, error), typed from the live schema
users, err := q.ListUsers(ctx, 10, 0)
err = q.WithTx(tx).DeleteUser(ctx, u.ID)
```

**Learn more:** [github.com/goloop/pgc](https://github.com/goloop/pgc) · [reference](https://pkg.go.dev/github.com/goloop/pgc)

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

## websocket

`websocket` implements the WebSocket protocol (RFC 6455) on top of the standard
library. It provides a server-side upgrade, a client-side dial, the
permessage-deflate extension and subprotocol negotiation.

A connection is a `Conn`. The server upgrade accepts same-origin requests by
default, which guards against cross-site hijacking; configure the allowed
origins explicitly when you need cross-origin clients.

```go
package main

import (
	"net/http"

	"github.com/goloop/websocket"
)

func echo(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Upgrade(w, r)
	if err != nil {
		return
	}
	defer ws.Close()

	for {
		mt, data, err := ws.ReadMessage()
		if err != nil {
			break
		}
		if err := ws.WriteMessage(mt, data); err != nil {
			break
		}
	}
}
```

**Learn more:** [github.com/goloop/websocket](https://github.com/goloop/websocket) · [reference](https://pkg.go.dev/github.com/goloop/websocket)

## app

`app` is a small lifecycle and composition kernel: it owns the start/stop
sequence of a service so your `main` does not repeat it. You register
components - anything with a `Name`, a non-blocking `Start` and a `Stop` - and
`app` starts them in order, waits for a signal, parent cancellation or a fatal
component error, then stops them in reverse order within a bounded timeout.

It is not a framework: no global state, no dependency-injection container, no
routing. Ready-made components cover the usual cases - `HTTPServer` wraps an
`*http.Server`, `Worker` wraps a blocking loop, `Closer` wraps a cleanup
function - and it exposes a read-only status snapshot that an observability
module can turn into a health check without either package importing the other.

```go
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/goloop/app"
)

func main() {
	a := app.New("api", app.WithShutdownTimeout(10*time.Second))
	a.Use(app.HTTPServer("http", &http.Server{Addr: ":8080", Handler: mux}))
	a.Use(app.Closer("db", pool.Close))

	if err := a.Run(context.Background()); err != nil {
		panic(err)
	}
}
```

**Learn more:** [github.com/goloop/app](https://github.com/goloop/app) · [reference](https://pkg.go.dev/github.com/goloop/app)

## observe

`observe` provides health, readiness and build-info endpoints for a service,
standard library only. It owns the operational-status model so every service
exposes the same `/healthz`, `/readyz` and build-info shape without copy-pasted
handlers. Health is process-level liveness and always ok; readiness runs the
registered dependency checks in parallel and fails when any one is unusable.

Each check is bounded by a per-check timeout that holds even when a check
ignores its context, a panicking check is recovered rather than crashing the
process, and error details are redacted by default so a health endpoint never
leaks hostnames or credentials. It never imports a lifecycle kernel: bridge to
one with a check over a status snapshot.

```go
package main

import (
	"context"
	"net/http"

	"github.com/goloop/observe"
)

func main() {
	obs := observe.New(observe.WithService("api"))
	obs.Check("postgres", func(ctx context.Context) error { return pool.Ping(ctx) })

	http.Handle("/healthz", obs.HealthHandler()) // 200 while the process lives
	http.Handle("/readyz", obs.ReadyHandler())   // 503 if a dependency is down
	http.ListenAndServe(":8080", nil)
}
```

**Learn more:** [github.com/goloop/observe](https://github.com/goloop/observe) · [reference](https://pkg.go.dev/github.com/goloop/observe)

## jwt

`jwt` issues and verifies compact JSON Web Tokens, deliberately limited to
HS256, standard library only. It follows the JWS compact serialization of
RFC 7515 and the registered claims of RFC 7519, but supports exactly one
algorithm with strict defaults: there is no algorithm negotiation, no `none`
and no asymmetric keys - a smaller surface is a safer surface.

The key must be at least 32 bytes; verification requires `alg=HS256` and a
present expiry, compares the HMAC with a constant-time check before reading the
payload, and rejects `crit` headers, non-strict base64url and mistyped
registered claims. Key rotation is built in: sign with the primary key, verify
against any configured key.

```go
package main

import (
	"fmt"
	"time"

	"github.com/goloop/jwt"
)

func main() {
	key := []byte("a-32-byte-or-longer-secret-key!!")

	token, _ := jwt.Sign(jwt.Claims{
		Subject:   "user-123",
		Issuer:    "api",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}, key)

	claims, err := jwt.Verify(token, key, jwt.WithIssuer("api"))
	fmt.Println(claims.Subject, err) // user-123 <nil>
}
```

**Learn more:** [github.com/goloop/jwt](https://github.com/goloop/jwt) · [reference](https://pkg.go.dev/github.com/goloop/jwt)

## auth

`auth` provides authentication primitives: password hashing, access tokens,
HTTP middleware and refresh-token rotation. It gives safe building blocks
without becoming an identity platform - there is no user repository, no RBAC
schema and no OAuth, so persistence and user management stay in your
application. It depends only on the standard library and its sibling `jwt`.

Passwords are hashed with a self-describing PBKDF2 encoding (the `PasswordHasher`
interface lets you swap the algorithm); access tokens are HS256 JWTs with
mandatory expiry and constant-time verification; the `Bearer` and `Cookie`
middleware authenticate a request, while `Require` and `RequireScope` enforce
presence and scope. Refresh tokens support rotation.

```go
package main

import (
	"net/http"

	"github.com/goloop/auth"
)

func main() {
	tm := auth.NewTokenManager(secret, auth.WithIssuer("api"))

	token, _ := tm.Issue(auth.Subject{ID: "user-1", Scopes: []string{"read"}})
	_ = token

	http.Handle("/me", tm.Bearer(auth.Require(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			sub, _ := auth.SubjectFrom(r.Context())
			w.Write([]byte(sub.ID))
		}))))
}
```

**Learn more:** [github.com/goloop/auth](https://github.com/goloop/auth) · [reference](https://pkg.go.dev/github.com/goloop/auth)

## argon2id

`argon2id` hashes and verifies passwords with Argon2id (RFC 9106), the
memory-hard function recommended for password storage. Memory-hard means each
hash costs a fixed amount of RAM, so a stolen hash is expensive to crack
offline. It implements Argon2id and its underlying BLAKE2b on the standard
library alone, with no dependencies, and is pinned to the official RFC test
vectors.

The encoded hash is a self-describing PHC string, and the `Hasher` method set
matches the `PasswordHasher` interface in `auth`, so it drops in there without
either package importing the other.

```go
package main

import "github.com/goloop/argon2id"

func main() {
	h := argon2id.New()

	encoded, _ := h.Hash([]byte("correct horse battery staple")) // store it
	_ = h.Verify(encoded, []byte("correct horse battery staple")) // nil on match
}
```

**Learn more:** [github.com/goloop/argon2id](https://github.com/goloop/argon2id) · [reference](https://pkg.go.dev/github.com/goloop/argon2id)

## session

`session` provides secure, signed cookie sessions for browser apps, standard
library only. The whole session lives in an HMAC-SHA256 signed cookie, so there
is no server-side store to run. It is a companion to token-based auth, not a
replacement: `session` owns cookie and browser state, while an auth package owns
subjects, passwords and tokens.

The cookie is HttpOnly, SameSite=Lax and signed; enable `Secure` in production.
The payload is versioned so the format can evolve without breaking old cookies,
and key rotation is built in. Read with `LoadOrNew` (or the middleware plus
`session.From`), then `Save` to persist.

```go
package main

import (
	"net/http"

	"github.com/goloop/session"
)

func main() {
	m := session.New(secret,
		session.WithName("sid"),
		session.WithSecure(true),
		session.WithTTL(24*time.Hour),
	)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s := m.LoadOrNew(r)
		s.Set("theme", "dark")
		_ = m.Save(w, s)
	})
}
```

**Learn more:** [github.com/goloop/session](https://github.com/goloop/session) · [reference](https://pkg.go.dev/github.com/goloop/session)

## mail

`mail` builds and sends outbound email, standard library only. It is a small
transport and message builder, not a marketing platform or a template engine:
it constructs an RFC 5322/MIME message and delivers it over SMTP, with drop-in
transports for development and tests. Text and/or HTML bodies, attachments,
validated addresses and RFC 2047-encoded display names are all handled.

A `Sender` delivers a message: `NewSMTP` sends over SMTP (STARTTLS by default),
`NewLogger` writes the built message to a writer for development, and
`NewCapture` records messages for tests - so the same code path runs in
production and under test.

```go
package main

import (
	"context"

	"github.com/goloop/mail"
)

func main() {
	sender := mail.NewSMTP(mail.SMTPConfig{
		Host: "smtp.example.com",
		From: mail.Address{Email: "no-reply@example.com"},
	})

	_ = sender.Send(context.Background(), mail.Message{
		To:      []mail.Address{{Email: "user@example.com"}},
		Subject: "Welcome",
		Text:    "Hello and welcome.",
		HTML:    "<p>Hello and welcome.</p>",
	})
}
```

**Learn more:** [github.com/goloop/mail](https://github.com/goloop/mail) · [reference](https://pkg.go.dev/github.com/goloop/mail)

## cli

`cli` is a command-line tool - not a library - that scaffolds and inspects
goloop-first Go projects. It is a small dispatcher over `os.Args` with no
third-party dependencies, and it never commits, pushes or deploys. Install it
with `go install github.com/goloop/cli@latest` (the command is `goloop`).

It scaffolds a minimal service (`goloop new`), reports which expected tools are
installed (`goloop doctor`), summarizes the module in the current directory
(`goloop status`), and scans a tree for stray editor/assistant artifacts
(`goloop check`).

```text
goloop new <name> [--module path] [--dir path] [--force]
goloop check [dir]     scan a project tree
goloop doctor          check that expected tools are installed
goloop status          summarize the module in the current directory
goloop version         print the CLI version
```

**Learn more:** [github.com/goloop/cli](https://github.com/goloop/cli) · [reference](https://pkg.go.dev/github.com/goloop/cli)

## How to choose

Use `env` and `opt` at program startup, `mux`, `middlewares`, `qp` and `resp`
in HTTP handlers, `websocket` for realtime connections, `ai` to talk to LLM
providers behind one interface, `pgc` to compile your SQL into typed Go
against PostgreSQL, `is` for validation, `log` for operational output, `set`
and `g` inside business logic, `key` for public reversible IDs, `kind` when a
parser or decoder needs to introspect types, `scs`, `slug` and `t13n` for
string processing, and `trit` whenever unknown state is a first-class value.

For the shape of a whole service, reach for the application layer: `app` to own
the start/stop lifecycle and graceful shutdown, `observe` for health and
readiness endpoints, `auth`, `jwt` and `session` for tokens, route protection
and browser sessions, and `mail` for outbound email. The `cli` command
scaffolds a new goloop-first project so you start from a working skeleton. See
[Building a service](#building-a-service) for how these layers fit together.

Each module is intentionally small. You do not need to adopt the whole group:
install only the module that closes the specific problem in front of you.
