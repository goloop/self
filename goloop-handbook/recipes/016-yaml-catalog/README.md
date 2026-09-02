# Recipe 016: a catalog people edit by hand

Part of the [GoLoop One handbook](../../main.md). Read the chapter:
[16. A catalog people edit by hand](../../chapters/16-yaml-catalog.md).

```sh
go test ./...   # pins defaults, required keys, merge, expansion and the export
go run .        # loads both documents and prints the redacted export
```

No environment is required.

## Two files, two decoders

A service usually reads more than one kind of YAML, and the difference is not
technical. One file is written by someone on the team, in a format we own. The
other arrives from outside, in a format someone else defined.

- **The catalog** is loaded with `WithStrict()`. A curator maintains it, so an
  unknown key is a typo in a known one, and skipping it means the edit never
  reaches the service.
- **The manifest** is loaded with plain `Unmarshal`. This struct is a subset of
  a format we do not own, so an unknown key is a field we do not need yet, not
  a reason to refuse someone's upload.

Strictness is a property of who wrote the file. Turning it on everywhere is as
wrong as leaving it off everywhere.

## What the tags say

- `,required` refuses a file that leaves a key out, and reports it with a line
  number. It replaces the `if s.Slug == "" { ... }` block that otherwise grows
  at the top of every loader.
- `def:"30s"` fills a key the curator did not need to think about.
- `time.Duration` and `url.URL` are read from the text a person writes, so
  `timeout: 30s` and `endpoint: https://...` mean what they look like.

`Validate` covers what a tag cannot: a tag speaks about one field, and rules
that span fields are the application's own.

## Three things the tests pin

- A **merge key** supplies a value, so the key counts as present and `required`
  is satisfied. Presence is decided after merges are expanded.
- An **explicit `null`** is not absence. Writing it is a decision, and a default
  that undid it would be undoing the curator.
- **`LoadError.Unwrap`** keeps the chain alive, so `errors.Is(err,
  yaml.ErrRequired)` still answers after the loader has added its own context.
  Formatting the cause into a string reads the same and quietly ends it.
