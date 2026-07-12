package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/goloop/ai"
)

// TestGenerateContract does one cheap Generate against each configured provider,
// asserting the shared ai.Client contract returns text. It skips when no key is
// set, so `go test` passes without credentials (and without spending tokens).
func TestGenerateContract(t *testing.T) {
	ps := configured()
	if len(ps) == 0 {
		t.Skip("set ANTHROPIC_API_KEY and/or OPENAI_API_KEY to test against the APIs")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for _, p := range ps {
		resp, err := p.client.Generate(ctx, &ai.Request{
			Model:     p.model,
			Messages:  []ai.Message{ai.UserText("Reply with the single word: ok")},
			MaxTokens: 5,
		})
		if err != nil {
			t.Fatalf("[%s] generate: %v", p.name, err)
		}
		if resp.Text() == "" {
			t.Fatalf("[%s] empty response", p.name)
		}
	}
}

var _ = os.Getenv
