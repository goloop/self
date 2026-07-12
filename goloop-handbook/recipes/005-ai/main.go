// Recipe 005: ask a language model, swap the provider.
//
// The task: call a large language model from Go, and keep the calling code the
// same whether you use Anthropic or OpenAI. goloop/ai is the provider-agnostic
// contract (Client, Request, Response, Chunk); each driver (anthropic, openai)
// implements it. Your code speaks ai.Client; only the constructor and the model
// name change.
//
// Three examples, all against the real APIs:
//
//	A. one question, every provider - the same *ai.Request through each Client;
//	B. streaming - tokens arrive incrementally over ai.Client.Stream;
//	C. shaping the answer - a System prompt and MaxTokens constrain the output.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/goloop/ai"
	"github.com/goloop/anthropic"
	"github.com/goloop/openai"
)

// provider pairs a name with an ai.Client and the model to use with it.
type provider struct {
	name   string
	model  string
	client ai.Client
}

// configured returns the providers whose API key is present, in a stable order.
func configured() []provider {
	var ps []provider
	if k := os.Getenv("ANTHROPIC_API_KEY"); k != "" {
		ps = append(ps, provider{"anthropic", anthropic.ModelClaudeHaiku45, anthropic.New(k)})
	}
	if k := os.Getenv("OPENAI_API_KEY"); k != "" {
		ps = append(ps, provider{"openai", openai.ModelGPT4oMini, openai.New(k)})
	}
	return ps
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "recipe:", err)
		os.Exit(1)
	}
}

func run() error {
	ps := configured()
	if len(ps) == 0 {
		return fmt.Errorf("set ANTHROPIC_API_KEY and/or OPENAI_API_KEY")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Example A: the same request, answered by every configured provider. The
	// code is identical per provider; only Model differs.
	fmt.Println("A. one question, every provider (Generate):")
	for _, p := range ps {
		req := &ai.Request{
			Model:     p.model,
			System:    "You are terse. Answer in one short sentence.",
			Messages:  []ai.Message{ai.UserText("What is a database index?")},
			MaxTokens: 80,
		}
		resp, err := p.client.Generate(ctx, req)
		if err != nil {
			fmt.Printf("   [%s] error: %v\n", p.name, err)
			continue
		}
		fmt.Printf("   [%s] in=%d out=%d: %s\n",
			p.name, resp.Usage.InputTokens, resp.Usage.OutputTokens, oneLine(resp.Text()))
	}

	// Example B: stream the answer from the first provider; tokens arrive as
	// they are produced.
	p := ps[0]
	fmt.Printf("B. streaming from %s (Stream):\n   ", p.name)
	stream := p.client.Stream(ctx, &ai.Request{
		Model:     p.model,
		Messages:  []ai.Message{ai.UserText("Name three Go standard library packages, comma-separated.")},
		MaxTokens: 60,
	})
	for chunk, err := range stream {
		if err != nil {
			fmt.Printf("\n   error: %v\n", err)
			break
		}
		fmt.Print(chunk.Text)
		if chunk.Done {
			fmt.Printf("\n   (done, out=%d tokens)\n", chunk.Usage.OutputTokens)
		}
	}

	// Example C: shape the answer. A System prompt plus a tiny MaxTokens force a
	// single word.
	fmt.Printf("C. shaping the answer with System + MaxTokens (%s):\n", p.name)
	resp, err := p.client.Generate(ctx, &ai.Request{
		Model:     p.model,
		System:    "Reply with exactly one word and nothing else.",
		Messages:  []ai.Message{ai.UserText("The capital of France?")},
		MaxTokens: 5,
	})
	if err != nil {
		return err
	}
	fmt.Printf("   -> %q\n", oneLine(resp.Text()))
	return nil
}

// oneLine collapses newlines so a multi-line answer prints on one report line.
func oneLine(s string) string {
	out := ""
	for _, r := range s {
		if r == '\n' || r == '\r' {
			out += " "
			continue
		}
		out += string(r)
	}
	return out
}
