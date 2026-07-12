[« Preface](00-preface.md) · [Contents](../main.md) · [JSON HTTP API »](02-http-json-api.md)

---

# 01. Configuration you can operate

**Task.** Read the service's settings from a `.env` file during local
development, let real environment variables win in production, and let
command-line flags win over both - with sensible defaults when nothing is set.
Keep secrets off the command line.

**Modules.** [`env`](https://github.com/goloop/env) reads `.env` files and the
environment into a struct; [`opt`](https://github.com/goloop/opt) parses
command-line flags into the *same* struct.

**Recipe.** [`recipes/001-configuration`](../recipes/001-configuration/)

## The idea

Configuration is the first thing every service needs and the first thing people
get wrong: a `.env` that must exist or the program crashes, flags that do not
line up with environment variables, secrets that end up in a shell history.

GoLoop treats the whole configuration as one plain struct. Each field says
where its value can come from, in three tags:

```go
type Config struct {
	Addr     string        `env:"APP_ADDR" def:":8080" opt:"addr" help:"listen address"`
	Env      string        `env:"APP_ENV" def:"dev" opt:"env" help:"dev or prod"`
	Timeout  time.Duration `env:"APP_TIMEOUT" def:"15s" opt:"timeout" help:"request timeout"`
	Debug    bool          `env:"APP_DEBUG" def:"false" opt:"debug" help:"verbose logging"`
	Secret   string        `env:"APP_SECRET" opt:"-"`
	Replicas int           `env:"APP_REPLICAS" def:"1" opt:"replicas" help:"worker count"`
}
```

- `env` names the environment variable.
- `def` is the built-in default.
- `opt` names the command-line flag; `opt:"-"` means "never a flag", which is
  how `Secret` stays out of `--help` and shell history.

The field types are ordinary Go types - `time.Duration` and `bool` and `int`
are parsed for you.

## Layering the sources

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

`env.LoadSafe` is the quiet hero here: `env.Load` fails if the named file is
missing, which forces every caller to guard it with `os.Stat`. `LoadSafe` skips
a missing file and still reports real parse errors, which is exactly the
"read `.env` if it is there" behavior you want.

## Execution report

Built, tested, and run four times to show the priority order (`go test ./...`
then `go run .` with different sources):

```text
$ go test ./...
ok  	goloop.one/handbook/001-configuration	0.002s

$ go run .                       # defaults only
addr=:8080 env=dev timeout=15s debug=false replicas=1 secret_set=false

$ printf 'APP_ENV=staging\nAPP_ADDR=:9090\nAPP_SECRET=from-dotenv\n' > .env
$ go run .                       # the .env file applies
addr=:9090 env=staging timeout=15s debug=false replicas=1 secret_set=true

$ APP_ENV=prod go run .          # a real env var overrides the .env value
addr=:9090 env=prod timeout=15s debug=false replicas=1 secret_set=true

$ APP_ENV=prod go run . --env qa --replicas 6 --debug   # a flag overrides all
addr=:9090 env=qa timeout=15s debug=true replicas=6 secret_set=true
```

Read the four runs top to bottom and you can see the ladder: defaults, then
`.env`, then the real environment beating the `.env`, then a flag beating
everything. `secret_set=true` from the `.env` onward, and `APP_SECRET` never
appears as a flag.

## What you learned

- Describe configuration **once**, as a struct with `env`/`def`/`opt` tags.
- Apply sources lowest-priority first: `env.LoadSafe` -> `env.Unmarshal` ->
  `opt.UnmarshalArgs`.
- Use `opt:"-"` for secrets so they are read from the environment only.
- `LoadSafe` makes the same binary run in dev and prod without a code change.

Next: serve something over HTTP.

---

[« Preface](00-preface.md) · [Contents](../main.md) · [JSON HTTP API »](02-http-json-api.md)
