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
system, no runtime you have to learn instead of Go. Each module is a normal Go
package that solves one ordinary problem and then gets out of the way. You
import the one you need. The standard library stays the vocabulary: `net/http`,
`database/sql`, `log/slog`, `context`, `encoding/json`. GoLoop adds only the
thin layer you would otherwise rewrite in every project.

## Why a recipe book

A reference manual is organized around the *code*: one page per package, one
entry per function. That is the right shape when you already know what you are
looking for. It is the wrong shape when you are new and you have a *task*.

This book is organized around the task. Each chapter names something you
actually need to do and answers it with a recipe: a small, complete program you
can read top to bottom, run, and adapt. The reference (`README.md`, each
module's `DOC.md`) is still there when you want the full surface of a package.
This book is the on-ramp.

## How the recipes are made

Every recipe is real. Before a chapter is written, its program is:

1. **built** - it compiles with the published module versions;
2. **tested** - it ships with `main_test.go`, and the tests pass;
3. **run** - it is deployed to a scratch directory and executed. When it needs
   a database, a real PostgreSQL is started; when it needs a model, a real API
   is called.

The output of that run is printed in the chapter as an **execution report**, so
you can see exactly what the code does, not what it is supposed to do. If a
recipe cannot be made to run cleanly, it does not go in the book.

## The one habit

There is a single habit that makes GoLoop feel light: **reach for the smallest
piece that solves the task, and let the standard library do the rest.**

You do not adopt "GoLoop" the way you adopt a framework. You adopt `env` for
configuration today, `mux` and `resp` for an endpoint tomorrow, `pgc` for the
database next week. Each one is a separate module with its own version. The
first recipe - configuration - shows the shape of this: a plain struct, two
small modules, and a program you could have written yourself, only shorter.

Turn the page.

---

[« Contents](../main.md) · [Contents](../main.md) · [Configuration »](01-configuration.md)
