package main

import (
	"errors"
	"testing"

	"github.com/goloop/env/v2"
)

// TestDefaults checks that an empty environment and no flags yield the
// documented defaults.
func TestDefaults(t *testing.T) {
	cfg, err := load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":8080" || cfg.Env != "dev" || cfg.Replicas != 1 || cfg.Debug {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.Timeout.String() != "15s" {
		t.Fatalf("timeout default: %s", cfg.Timeout)
	}
}

// TestEnvThenFlags checks the priority order: environment sets a value, a flag
// overrides it, and a secret is read from the environment only.
func TestEnvThenFlags(t *testing.T) {
	t.Setenv("APP_ENV", "prod")
	t.Setenv("APP_REPLICAS", "4")
	t.Setenv("APP_SECRET", "s3cr3t")

	cfg, err := load([]string{"--replicas", "8", "--debug"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != "prod" {
		t.Errorf("env from environment = %q, want prod", cfg.Env)
	}
	if cfg.Replicas != 8 {
		t.Errorf("replicas: flag should win, got %d", cfg.Replicas)
	}
	if !cfg.Debug {
		t.Error("debug flag not applied")
	}
	if cfg.Secret != "s3cr3t" {
		t.Errorf("secret from env = %q", cfg.Secret)
	}
}

// TestOriginsSlice checks example A's slice field: one variable split on the
// sep tag becomes a []string.
func TestOriginsSlice(t *testing.T) {
	t.Setenv("APP_ORIGINS", "https://a.example,https://b.example")
	cfg, err := load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Origins) != 2 || cfg.Origins[0] != "https://a.example" {
		t.Fatalf("origins = %v, want two split values", cfg.Origins)
	}
}

// TestRequiredField checks example D: a required variable is an error when
// missing and succeeds when set.
func TestRequiredField(t *testing.T) {
	var need mustHave
	if err := env.Unmarshal(&need); !errors.Is(err, env.ErrRequired) {
		t.Fatalf("missing required -> %v, want ErrRequired", err)
	}
	t.Setenv("DATABASE_URL", "postgres://localhost/app")
	if err := env.Unmarshal(&need); err != nil || need.DatabaseURL == "" {
		t.Fatalf("present required -> %v %q", err, need.DatabaseURL)
	}
}

// TestParseSnippet checks example B: parsing .env text into a map.
func TestParseSnippet(t *testing.T) {
	m, err := envParse("HOST=db\nPORT=5432\n# note\nTAGS=a,b\n")
	if err != nil {
		t.Fatal(err)
	}
	if m["HOST"] != "db" || m["PORT"] != "5432" || m["TAGS"] != "a,b" {
		t.Fatalf("parsed map = %v", m)
	}
}

// TestMarshalRoundTrip checks example C: writing a struct back to .env lines
// that parse into the same values.
func TestMarshalRoundTrip(t *testing.T) {
	in := Config{Addr: ":9000", Env: "prod", Replicas: 3, Debug: true}
	text := marshalEnv(in)
	m, err := envParse(text)
	if err != nil {
		t.Fatal(err)
	}
	if m["APP_ADDR"] != ":9000" || m["APP_ENV"] != "prod" || m["APP_REPLICAS"] != "3" {
		t.Fatalf("round-trip lost values: %q -> %v", text, m)
	}
}
