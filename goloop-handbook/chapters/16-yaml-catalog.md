[« Errors a frontend can switch on](15-error-slugs.md) · [Contents](../main.md)

---

# 16. A catalog people edit by hand

**Task.** A service reads more than one kind of YAML, and the kinds are not
alike. One file is a curated catalog: someone on the team edits it, reviews it,
and expects a mistake in it to be caught. Another arrives as an upload, written
by a tool the team does not control, in a format of which our struct is only a
subset. Reading both with the same call is how a curator's typo silently
disappears and a user's harmless extra key gets their import rejected.

**Modules.** [`yaml`](https://github.com/goloop/yaml) reads the configuration
subset of YAML with the `encoding/json` mapping, and takes options that say
which of these two files it is looking at.

**Recipe.** [`recipes/016-yaml-catalog`](../recipes/016-yaml-catalog/)

## Strictness is a property of the author

The catalog is loaded strictly:

```go
err := yaml.Unmarshal(data, &c,
	yaml.WithStrict(),       // an unknown key is a typo
	yaml.WithExpandStrict(), // a token comes from the environment
)
```

The upload is not:

```go
err := yaml.Unmarshal(data, &m) // an unknown key is a field we do not need yet
```

Both are right, and each would be wrong in the other's place. `WithStrict` asks
"did the author mean something this program does not know?" - which is a useful
question about a file a colleague wrote and a rude one about a file someone
else's tool produced. There is no default that is correct for both, which is
why it is a decision at the call and not a setting on the package.

Note what the upload does *not* get: expansion. The file came from outside, and
letting it read this process's environment is not a feature.

## What a tag can say

Two loaders in production had grown the same block at the top:

```go
if fm.Title == "" || fm.Slug == "" || fm.Language == "" || fm.Category == "" {
	return draft{}, fmt.Errorf("%s: title, slug, language and category are all required", path)
}
```

It works, and it tells the author almost nothing: which one, and where. The
struct can say it instead, once, where the field is declared:

```go
type Source struct {
	Slug     string        `yaml:"slug,required"`
	Endpoint url.URL       `yaml:"endpoint,required"`
	Timeout  time.Duration `yaml:"timeout" def:"30s"`
	Quality  int           `yaml:"quality" def:"80"`
	Enabled  bool          `yaml:"enabled" def:"true"`
}
```

```
catalog: line 3: incomplete: main.Source: key "slug" is required and the
document does not set it
```

`def` is the other half. A curator should not have to write down the timeout
they agree with, and a loader should not have to carry `if s.Timeout == 0 { s.Timeout = 30 * time.Second }`
to make up for it.

The types pull their weight too. `timeout: 30s` is a `time.Duration` because
the field is one, and `timeout: 30` is refused rather than read as thirty
nanoseconds. `endpoint: https://example.com/feed.xml` is a `url.URL`, parsed at
load time rather than at the first request that uses it.

## Three answers worth knowing before you need them

**A merge key counts as present.** Curators reuse blocks, and they should:

```yaml
defaults: &defaults
  timeout: 45s
sources:
  - <<: *defaults
    slug: weekly
```

Presence is decided after merges are expanded, so `required` sees what the
decoder sees. If it did not, a file that reads perfectly would fail to load.

**An anchor holder is a key like any other.** YAML has no notion of a block
that exists only to be referenced, so under `WithStrict` the `defaults:` above
has to be declared:

```go
Defaults map[string]any `yaml:"defaults,omitempty"`
```

Nothing reads it. It is there because the first curator to reach for an anchor
would otherwise be told their file is wrong.

**An explicit `null` is not absence.**

```yaml
quality: null   # with def:"80" -> 0, not 80
```

Writing `null` is a decision. A default that undid it would be overruling the
person who wrote the file, and `required` is satisfied by the key being there,
because it asks whether the author addressed the setting, not whether the value
is non-empty.

## Keep the error chain alive

The loader wants to add context, and the obvious way throws away what it is
adding context to:

```go
return nil, fmt.Errorf("catalog%s", describe(err)) // the cause is now a string
```

After that, `errors.Is(err, yaml.ErrRequired)` is false and the caller who
wanted to tell "the file is incomplete" from "the file is wrong" is left
matching on message text. A small type keeps both:

```go
type LoadError struct {
	What string
	Err  error
}

func (e *LoadError) Error() string { return e.What + describe(e.Err) }
func (e *LoadError) Unwrap() error { return e.Err }
```

Now the message still carries the line number, and `errors.Is(err,
yaml.ErrRequired)` still answers.

## Writing one back out

An export is a file that gets attached to a ticket or committed to a branch, so
the tokens that came from the environment on the way in do not go out with it:

```go
out := *c
out.Sources = slices.Clone(c.Sources)
for i := range out.Sources {
	out.Sources[i].Token = "" // omitempty then leaves the key out entirely
}
return yaml.Marshal(out)
```

The output is deterministic - map keys sorted, struct fields in declaration
order - so a generated catalog does not churn in version control, and a
`time.Duration` goes back out as `45s` rather than as a number nobody can read.

## Takeaways

- Decide strictness per document, by who wrote it. `WithStrict` for a file a
  colleague maintains, plain `Unmarshal` for a format you do not own.
- Do not expand the environment into a file that came from outside.
- `,required` and `def` say in the struct what loaders otherwise say in `if`
  blocks, and they say it with a line number.
- Presence is decided after merge keys; an explicit `null` is a decision, not
  an omission.
- Wrap decode errors with a type that has `Unwrap`, or `errors.Is` stops
  working one line later.
- `Validate` still exists, for the rules that span more than one field.

This closes Part IV. You have taken the modules of Parts I-III and shaped them
the way production shapes them: fail early, refuse honestly, resolve the real
client, ask what a provider can do, answer in a vocabulary a frontend can act
on, and read a file the way its author meant it.

---

[« Errors a frontend can switch on](15-error-slugs.md) · [Contents](../main.md)
