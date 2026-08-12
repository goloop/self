# GoLoop One - a recipe book for Go APIs

📖 **English** · [Українська](main.uk.md)

*One solution for Go APIs, one task at a time.*

Welcome. This book helps you build a Go API with GoLoop, starting from what you
actually need to do: read configuration, answer JSON over HTTP, clean a signup
form, store rows in PostgreSQL, ask a language model. For each task you get a
recipe you can read, run and adapt.

Every recipe is a complete program in [`recipes/`](recipes/), with its own
`go.mod`, `main.go` and `main_test.go`. The chapter shows the real output of
running it, so you see exactly what the code does.

## How to use this book

- Read a chapter for the idea; open its recipe directory for the whole program.
- To run a recipe yourself:

  ```sh
  cd recipes/<name>
  go test ./...
  go run .
  ```

- The chapters are ordered but independent. Part I takes one module at a time;
  the later parts combine them into a service.
- You adopt one recipe without adopting the rest: every GoLoop module is a
  separate, [stdlib-first](../README.md) package with its own version.

## Contents

### Part I - One module at a time

- [01. Configuration you can operate](chapters/01-configuration.md) - `env`, `opt`
  · recipe [`001-configuration`](recipes/001-configuration/)
- [02. A JSON HTTP API without a framework](chapters/02-http-json-api.md) -
  `mux`, `resp`, `middlewares` · recipe [`002-http-json-api`](recipes/002-http-json-api/)
- [03. Validate and clean user input](chapters/03-validate-and-clean.md) -
  `is`, `norm` · recipe [`003-validate-and-clean`](recipes/003-validate-and-clean/)

### Part II - Data and models

- [04. Typed PostgreSQL with migrations](chapters/04-postgresql.md) - `pgc`
  · recipe [`004-postgresql`](recipes/004-postgresql/)
- [05. Ask a language model, swap the provider](chapters/05-ai.md) - `ai`, `anthropic`, `openai`
  · recipe [`005-ai`](recipes/005-ai/)
- [06. Sessions, tokens and passwords](chapters/06-auth.md) - `auth`, `jwt`,
  `session`, `key` · recipe [`006-auth`](recipes/006-auth/)
- [07. Slugs, transliteration and cases](chapters/07-slug.md) - `slug`, `t13n`,
  `scs` · recipe [`007-slug`](recipes/007-slug/)

### Part III - The whole stack

- [08. A service lifecycle](chapters/08-lifecycle.md) - `app`, `observe`, `log`
  · recipe [`008-lifecycle`](recipes/008-lifecycle/)
- [09. Real-time with WebSockets](chapters/09-websocket.md) - `websocket`
  · recipe [`009-websocket`](recipes/009-websocket/)
- [10. Putting it together](chapters/10-whole-stack.md) - one API on the whole
  GoLoop stack · recipe [`010-whole-stack`](recipes/010-whole-stack/)

### Part IV - Field patterns

Patterns drawn from services running the whole GoLoop stack in production. These
build on the modules of Parts I-III rather than introducing new ones.

- [11. A sign-in endpoint that survives production](chapters/11-hardened-signin.md) -
  `env`, `jwt`, `argon2id`, `auth`, `middlewares`, `mux`, `resp`
  · recipe [`011-hardened-signin`](recipes/011-hardened-signin/)
- [12. The refresh-token lifecycle](chapters/12-refresh-lifecycle.md) - `auth`,
  `auth/authtest` · recipe [`012-refresh-lifecycle`](recipes/012-refresh-lifecycle/)
- [13. Behind a reverse proxy](chapters/13-reverse-proxy.md) - `middlewares`
  (RealIP, RateLimit, context logger) · recipe [`013-reverse-proxy`](recipes/013-reverse-proxy/)
- [14. AI, production shape](chapters/14-ai-production.md) - `ai`, `anthropic`,
  `openai` · recipe [`014-ai-production`](recipes/014-ai-production/)
- [15. Errors a frontend can switch on](chapters/15-error-slugs.md) - `resp`,
  `qp`, `mux` · recipe [`015-error-slugs`](recipes/015-error-slugs/)

## Start here

New to GoLoop? Begin with the [preface](chapters/00-preface.md), then read Part I
in order. Already know what you need? Jump straight to the chapter.

---

*GoLoop is a group of small, focused Go modules. See the
[project README](../README.md) for the full module list and the design rules.*
