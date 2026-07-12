package main

import "testing"

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
