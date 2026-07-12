// Recipe 001: configuration that a person can actually operate.
//
// The task: read settings from a .env file for local development, let real
// environment variables win in production, and let command-line flags win over
// both - with sane defaults when nothing is set. This is the first thing every
// service needs, and goloop does it with two small modules and a plain struct.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/goloop/env/v2"
	"github.com/goloop/opt/v2"
)

// Config is the whole configuration of the service, described once as a struct.
// Each field carries three tags: env (environment variable name), def (default
// value) and opt (command-line flag). A field with opt:"-" is never a flag,
// which is how secrets stay off the command line.
type Config struct {
	Addr     string        `env:"APP_ADDR" def:":8080" opt:"addr" help:"listen address"`
	Env      string        `env:"APP_ENV" def:"dev" opt:"env" help:"dev or prod"`
	Timeout  time.Duration `env:"APP_TIMEOUT" def:"15s" opt:"timeout" help:"request timeout"`
	Debug    bool          `env:"APP_DEBUG" def:"false" opt:"debug" help:"verbose logging"`
	Secret   string        `env:"APP_SECRET" opt:"-"`
	Replicas int           `env:"APP_REPLICAS" def:"1" opt:"replicas" help:"worker count"`
}

func main() {
	cfg, err := load(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	fmt.Printf("addr=%s env=%s timeout=%s debug=%v replicas=%d secret_set=%v\n",
		cfg.Addr, cfg.Env, cfg.Timeout, cfg.Debug, cfg.Replicas, cfg.Secret != "")
}

// load builds the configuration by layering three sources, lowest priority
// first: built-in defaults, then the .env file and the process environment,
// then the command-line flags.
func load(args []string) (Config, error) {
	// LoadSafe reads .env when it is present and does nothing when it is not,
	// so the same binary runs in local development (with a file) and in
	// production (with real environment variables) without a code change.
	if err := env.LoadSafe(); err != nil {
		return Config{}, fmt.Errorf("loading .env: %w", err)
	}

	var cfg Config
	// Unmarshal fills the struct from defaults and the environment.
	if err := env.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("environment: %w", err)
	}
	// UnmarshalArgs lets flags override what the environment set.
	if err := opt.UnmarshalArgs(&cfg, args); err != nil {
		return Config{}, fmt.Errorf("flags: %w", err)
	}
	return cfg, nil
}
