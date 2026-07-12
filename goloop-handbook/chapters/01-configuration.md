[« Preface](00-preface.md) · [Contents](../main.md) · [JSON HTTP API »](02-http-json-api.md)

---

# 01. Configuration you can operate

**Task.** Read the service's settings from a `.env` file during local
development, let real environment variables win in production, and let flags win
over both, with sensible defaults when nothing is set. Keep secrets off the
command line.

**Modules.** [`env`](https://github.com/goloop/env) reads `.env` files and the
environment into a struct; [`opt`](https://github.com/goloop/opt) parses flags
into the *same* struct.

**Recipe.** [`recipes/001-configuration`](../recipes/001-configuration/)

## The configuration struct

Describe the whole configuration once, as a plain struct. Each field says where
its value can come from, in three tags:

```go
type Config struct {
	Addr     string        `env:"APP_ADDR" def:":8080" opt:"addr" help:"listen address"`
	Env      string        `env:"APP_ENV" def:"dev" opt:"env" help:"dev or prod"`
	Timeout  time.Duration `env:"APP_TIMEOUT" def:"15s" opt:"timeout" help:"request timeout"`
	Debug    bool          `env:"APP_DEBUG" def:"false" opt:"debug" help:"verbose logging"`
	Secret   string        `env:"APP_SECRET" opt:"-"`
	Replicas int           `env:"APP_REPLICAS" def:"1" opt:"replicas" help:"worker count"`
	Origins  []string      `env:"APP_ORIGINS" def:"http://localhost:3000" sep:"," opt:"origins"`
}
```

`env` names the environment variable, `def` is the default, `opt` names the
flag. `opt:"-"` means "never a flag", which keeps `Secret` out of `--help` and
shell history. The `sep` tag splits one variable into a slice, so
`APP_ORIGINS="https://a.example,https://b.example"` becomes a `[]string`. The
field types are ordinary Go types, parsed for you.

## Example A - layer the sources

Priority is just the order you apply the sources, lowest first:

```go
func load(args []string) (Config, error) {
	// LoadSafe reads .env when present and does nothing when absent, so the
	// same binary runs locally (with a file) and in production (without one).
	if err := env.LoadSafe(); err != nil {
		return Config{}, fmt.Errorf("loading .env: %w", err)
	}
	var cfg Config
	if err := env.Unmarshal(&cfg); err != nil { // defaults + environment
		return Config{}, fmt.Errorf("environment: %w", err)
	}
	if err := opt.UnmarshalArgs(&cfg, args); err != nil { // flags win
		return Config{}, fmt.Errorf("flags: %w", err)
	}
	return cfg, nil
}
```

`env.LoadSafe` is the quiet hero: `env.Load` fails when the named file is
missing, which forces every caller to guard it. `LoadSafe` skips a missing file
and still reports real parse errors, which is exactly the "read `.env` if it is
there" behavior you want.

## Example B - parse a snippet into a map

Sometimes you do not want a struct, only the key/value pairs. `env.Parse` reads
`.env` text from any `io.Reader` into a `map[string]string`:

```go
m, _ := env.Parse(strings.NewReader("HOST=db.internal\nPORT=5432\n# a comment\nTAGS=a,b,c\n"))
// m["HOST"] == "db.internal", comments ignored
```

## Example C - write the struct back out

`env.MarshalWriter` turns a struct back into `.env` lines, handy for generating
a template. Note it writes *every* field, including `Secret`, so redact secrets
before sharing a generated file.

```go
var b strings.Builder
_ = env.MarshalWriter(&b, cfg) // APP_ADDR=:8080\nAPP_ENV=...\n
```

## Example D - make a field required

A missing setting should fail loudly at startup, not surface as a mysterious
zero value later. Add `required` to the `env` tag: with no default and no value
set, `env.Unmarshal` returns `env.ErrRequired`, naming the key:

```go
type mustHave struct {
	DatabaseURL string `env:"DATABASE_URL,required"`
}

var need mustHave
err := env.Unmarshal(&need)              // missing -> env.ErrRequired
errors.Is(err, env.ErrRequired)          // true, and the message names DATABASE_URL

os.Setenv("DATABASE_URL", "postgres://localhost/app")
_ = env.Unmarshal(&need)                 // now nil; need.DatabaseURL is set
```

Use `required` for the values a service genuinely cannot start without, and a
`def` for everything that has a sane fallback.

## Execution report

Tested, then run once with a `.env` present and a flag set:

```text
$ go test ./...
ok  	goloop.one/handbook/001-configuration	0.002s

$ printf 'APP_ENV=staging\nAPP_SECRET=from-dotenv\n' > .env
$ go run . --replicas 4
A. layered config (defaults < .env/env < flags):
   addr=:8080 env=staging timeout=15s debug=false replicas=4 secret_set=true
   origins=[http://localhost:3000] (1, one variable split on ",")
B. parse a .env snippet into a map (no struct):
   HOST=db.internal PORT=5432 TAGS=a,b,c
C. marshal the struct back to .env lines (a template):
   APP_ADDR=:8080
   APP_ENV=staging
   APP_TIMEOUT=15s
   APP_DEBUG=false
   APP_SECRET=from-dotenv
   APP_REPLICAS=4
   APP_ORIGINS=http://localhost:3000
D. a required field (env:"...,required"):
   missing -> error: true
   present -> ok=true value=postgres://localhost/app
```

In A, `env=staging` came from the `.env`, `replicas=4` from the flag, `origins`
was the single default value, and the secret was read but never printed. In C
the same struct round-trips to `.env` lines (and yes, `APP_SECRET` is in there,
which is the reminder to redact). In D the required key errors when unset and
succeeds once provided.

## What you learned

- Describe configuration **once**, as a struct with `env`/`def`/`opt` tags.
- Apply sources lowest-priority first: `env.LoadSafe` -> `env.Unmarshal` ->
  `opt.UnmarshalArgs`; use `opt:"-"` for secrets.
- A `sep` tag splits one variable into a slice; `env:"NAME,required"` makes a
  value mandatory and returns `env.ErrRequired` when it is missing.
- `env.Parse` reads a snippet into a map when you do not want a struct.
- `env.MarshalWriter` writes a struct back to `.env` lines (redact secrets).

Next: serve something over HTTP.

---

[« Preface](00-preface.md) · [Contents](../main.md) · [JSON HTTP API »](02-http-json-api.md)
