[« Contents](../main.md) · [Contents](../main.md) · [Configuration »](01-configuration.md)

---

# Preface

## What GoLoop is

GoLoop is a group of small, focused Go modules for the everyday work of a
network service: reading configuration, parsing flags, validating input,
routing requests, writing responses, generating typed SQL, talking to language
models, and then running the whole thing with a lifecycle, health checks,
sessions and email.

It is not a framework. There is no `New(app)` that owns your `main`, no plugin
system, no runtime to learn instead of Go. Each module is a normal Go package
that solves one ordinary problem and then gets out of the way. You import the
one you need. The standard library stays the vocabulary: `net/http`,
`database/sql`, `log/slog`, `context`, `encoding/json`. GoLoop adds only the
thin layer you would otherwise rewrite in every project.

## How to read this book

Each chapter names a task you actually have and answers it with a recipe: a
small, complete program you can read top to bottom, run, and adapt. You will see
two or three uses of each module, so you leave the chapter knowing its range,
not just its simplest call.

Read Part I in order; it takes one module at a time and builds up. The later
parts combine those pieces into a working service. When you want the full
surface of a package, the reference is a link away: the
[project README](../README.md) and each module's `DOC.md`.

## The one habit

There is a single habit that makes GoLoop feel light: reach for the smallest
piece that solves the task, and let the standard library do the rest.

You do not adopt "GoLoop" the way you adopt a framework. You adopt `env` for
configuration today, `mux` and `resp` for an endpoint tomorrow, `pgc` for the
database next week. Each one is a separate module with its own version. The
first recipe, configuration, shows the shape of this: a plain struct, two small
modules, and a program you could have written yourself, only shorter.

Turn the page.

---

[« Contents](../main.md) · [Contents](../main.md) · [Configuration »](01-configuration.md)
