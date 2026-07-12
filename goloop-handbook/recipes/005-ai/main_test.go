package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/goloop/ai"
	"github.com/goloop/anthropic"
	"github.com/goloop/openai"
)

// testClients constructs a client per configured provider, or nil to skip.
func testClients() []namedClient {
	var cs []namedClient
	if k := os.Getenv("ANTHROPIC_API_KEY"); k != "" {
		cs = append(cs, namedClient{"anthropic", anthropic.ModelClaudeHaiku45, anthropic.New(k)})
	}
	if k := os.Getenv("OPENAI_API_KEY"); k != "" {
		cs = append(cs, namedClient{"openai", openai.ModelGPT4oMini, openai.New(k)})
	}
	return cs
}

// TestGenerate does one cheap Generate per provider, asserting the shared
// ai.Client contract returns text. It skips when no key is set.
func TestGenerate(t *testing.T) {
	cs := testClients()
	if len(cs) == 0 {
		t.Skip("set ANTHROPIC_API_KEY and/or OPENAI_API_KEY to test against the APIs")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for _, c := range cs {
		resp, err := c.client.Generate(ctx, &ai.Request{
			Model: c.model, MaxTokens: 5,
			Messages: []ai.Message{ai.UserText("Reply with the single word: ok")},
		})
		if err != nil {
			t.Fatalf("[%s] generate: %v", c.name, err)
		}
		if resp.Text() == "" {
			t.Fatalf("[%s] empty response", c.name)
		}
	}
}

// TestVision sends an image and checks the model can read it, exercising the
// multimodal message path. It runs on the first configured provider.
func TestVision(t *testing.T) {
	cs := testClients()
	if len(cs) == 0 {
		t.Skip("set a provider key to test vision")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := cs[0].client.Generate(ctx, &ai.Request{
		Model: cs[0].model, MaxTokens: 20,
		Messages: []ai.Message{{Role: ai.RoleUser, Parts: []ai.Part{
			ai.Text{Text: "What color is this square? One word."},
			ai.Image{MIME: "image/png", Data: redSquarePNG()},
		}}},
	})
	if err != nil {
		t.Fatalf("vision: %v", err)
	}
	if resp.Text() == "" {
		t.Fatal("empty vision response")
	}
}
