// Recipe 001: configuration that a person can actually operate.
//
// Four examples, from the everyday to the useful extras:
//
//	A. layered config - defaults, then .env and the environment, then flags,
//	                    including a comma-separated list parsed into a []string;
//	B. read a snippet - parse .env text into a map without a struct;
//	C. write it back  - marshal a struct out as .env lines (a template);
//	D. required field - a missing required variable is a clear, early error.
//
// The modules: env reads .env files and the environment; opt parses flags into
// the same struct.
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
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
	// A slice field: the sep tag splits one variable into many values, so
	// APP_ORIGINS="https://a.example,https://b.example" becomes a []string.
	Origins []string `env:"APP_ORIGINS" def:"http://localhost:3000" sep:"," opt:"origins" help:"allowed origins"`
}

// mustHave is a second, tiny config used only to show a required field. The
// env:"...,required" flag makes the variable mandatory: without a default and
// without the variable set, Unmarshal returns env.ErrRequired.
type mustHave struct {
	DatabaseURL string `env:"DATABASE_URL,required"`
}

func main() {
	cfg, err := load(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}

	fmt.Println("A. layered config (defaults < .env/env < flags):")
	fmt.Printf("   addr=%s env=%s timeout=%s debug=%v replicas=%d secret_set=%v\n",
		cfg.Addr, cfg.Env, cfg.Timeout, cfg.Debug, cfg.Replicas, cfg.Secret != "")
	fmt.Printf("   origins=%v (%d, one variable split on \",\")\n", cfg.Origins, len(cfg.Origins))

	fmt.Println("B. parse a .env snippet into a map (no struct):")
	m, _ := env.Parse(strings.NewReader("HOST=db.internal\nPORT=5432\n# a comment\nTAGS=a,b,c\n"))
	fmt.Printf("   HOST=%s PORT=%s TAGS=%s\n", m["HOST"], m["PORT"], m["TAGS"])

	fmt.Println("C. marshal the struct back to .env lines (a template):")
	var b strings.Builder
	_ = env.MarshalWriter(&b, cfg)
	for _, line := range strings.Split(strings.TrimSpace(b.String()), "\n") {
		fmt.Printf("   %s\n", line)
	}

	fmt.Println("D. a required field (env:\"...,required\"):")
	_ = os.Unsetenv("DATABASE_URL")
	var need mustHave
	errMissing := env.Unmarshal(&need)
	fmt.Printf("   missing -> error: %v\n", errMissing != nil && errors.Is(errMissing, env.ErrRequired))
	_ = os.Setenv("DATABASE_URL", "postgres://localhost/app")
	errPresent := env.Unmarshal(&need)
	fmt.Printf("   present -> ok=%v value=%s\n", errPresent == nil, need.DatabaseURL)
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
