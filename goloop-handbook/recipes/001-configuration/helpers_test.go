package main

import (
	"strings"

	"github.com/goloop/env/v2"
)

func envParse(s string) (map[string]string, error) { return env.Parse(strings.NewReader(s)) }

func marshalEnv(c Config) string {
	var b strings.Builder
	_ = env.MarshalWriter(&b, c)
	return b.String()
}
