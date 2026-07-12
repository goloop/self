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

### Part II - Data and models *(in progress)*

- 04. Typed PostgreSQL with migrations - `pgc`
- 05. Ask a language model, swap the provider - `ai`, `anthropic`, `openai`
- 06. Sessions, tokens and passwords - `auth`, `jwt`, `session`, `key`
- 07. Slugs, transliteration and cases - `slug`, `t13n`, `scs`

### Part III - The whole stack *(planned)*

- 08. A service lifecycle - `app`, `observe`, `log`
- 09. Real-time with WebSockets - `websocket`
- 10. Putting it together - one API on the whole GoLoop stack

## Start here

New to GoLoop? Begin with the [preface](chapters/00-preface.md), then read Part I
in order. Already know what you need? Jump straight to the chapter.

---

*GoLoop is a group of small, focused Go modules. See the
[project README](../README.md) for the full module list and the design rules.*
