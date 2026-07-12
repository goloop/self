# GoLoop One - a recipe book for Go APIs

📖 **English** · [Українська](main.uk.md)

*One solution for Go APIs, one task at a time.*

This is a book, not a reference manual. A reference tells you what every function
does; this book starts from a task a person actually has - "read my
configuration", "answer JSON over HTTP", "clean a signup form", "store rows in
PostgreSQL", "ask a model a question" - and shows a working recipe that solves
it with GoLoop.

Every recipe in this book is a complete, runnable Go program in
[`recipes/`](recipes/). Each one is built, tested and run before it is written
about, and the real output of that run is printed in the chapter as an
**execution report**. When a recipe needs a database or a language model, the
report is produced against a real one. Nothing here is pseudo-code.

## How to read this book

- Read a chapter for the idea; open its recipe directory for the whole program.
- Every recipe directory has a `go.mod`, a `main.go` and a `main_test.go`. To
  run one yourself: `cd recipes/<name> && go test ./... && go run .`
- Chapters are ordered, but independent. Each starts from foundations and
  builds up. The last part assembles the individual pieces into one service.
- Every module GoLoop offers is
  [stdlib-first and mostly zero-dependency](../README.md): you adopt one recipe
  without adopting the rest.

## Contents

### Part I - One module at a time

Each chapter solves one ordinary problem with the smallest set of modules.

- [01. Configuration you can operate](chapters/01-configuration.md) - `env`, `opt`
  · recipe [`001-configuration`](recipes/001-configuration/)
- [02. A JSON HTTP API without a framework](chapters/02-http-json-api.md) -
  `mux`, `resp`, `middlewares` · recipe [`002-http-json-api`](recipes/002-http-json-api/)
- [03. Validate and clean user input](chapters/03-validate-and-clean.md) -
  `is`, `norm` · recipe [`003-validate-and-clean`](recipes/003-validate-and-clean/)

### Part II - Data and models *(planned)*

- 04. Typed PostgreSQL with migrations - `pgc`
- 05. Ask a language model, swap the provider - `ai`, `anthropic`, `openai`
- 06. Sessions, tokens and passwords - `auth`, `jwt`, `session`, `key`
- 07. Slugs, transliteration and cases - `slug`, `t13n`, `scs`

### Part III - The whole stack *(planned)*

- 08. A service lifecycle - `app`, `observe`, `log`
- 09. Real-time with WebSockets - `websocket`
- 10. Putting it together - one API on the whole GoLoop stack

Part I is written and verified. Parts II and III are on the way; the recipes
that back them are being built and tested the same way, one at a time.

## The preface

New here? Start with the [preface](chapters/00-preface.md): what GoLoop is, why
a recipe book, and the one habit that makes the rest of it click.

---

*GoLoop is a group of small, focused Go modules. See the
[project README](../README.md) for the full module list and the design rules.*
