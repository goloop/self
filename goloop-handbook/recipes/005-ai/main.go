// Recipe 005: talk to a language model - text, streaming, photos and tools.
//
// goloop/ai is the provider-agnostic contract (Client, Request, Response,
// Chunk, and the content Parts: Text, Image, ToolUse, ToolResult). Each driver
// implements it: anthropic.New(key) and openai.New(key) both return an
// ai.Client, so the calling code is the same and only the model name differs.
//
// Examples, all against the real APIs:
//
//	A. construct + ask - anthropic.New / openai.New, then Generate;
//	B. stream          - tokens arrive incrementally over Stream;
//	C. shape           - a System prompt and MaxTokens constrain the reply;
//	D. photo (vision)  - send image bytes in a message and ask about them;
//	E. tools           - the model calls a function; you answer and it replies.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"time"

	"github.com/goloop/ai"
	"github.com/goloop/anthropic"
	"github.com/goloop/openai"
)

// namedClient pairs a display name and model with an ai.Client.
type namedClient struct {
	name   string
	model  string
	client ai.Client
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "recipe:", err)
		os.Exit(1)
	}
}

func run() error {
	// Construct a client per provider whose key is set. anthropic.New and
	// openai.New both return an ai.Client - this is the only provider-specific
	// code in the whole program.
	var clients []namedClient
	if k := os.Getenv("ANTHROPIC_API_KEY"); k != "" {
		clients = append(clients, namedClient{"anthropic", anthropic.ModelClaudeHaiku45, anthropic.New(k)})
	}
	if k := os.Getenv("OPENAI_API_KEY"); k != "" {
		clients = append(clients, namedClient{"openai", openai.ModelGPT4oMini, openai.New(k)})
	}
	if len(clients) == 0 {
		return fmt.Errorf("set ANTHROPIC_API_KEY and/or OPENAI_API_KEY")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	p := clients[0] // the default provider for the single-provider examples

	// A. construct + ask: the same request through every provider.
	fmt.Println("A. construct + ask (anthropic.New / openai.New, Generate):")
	for _, c := range clients {
		resp, err := c.client.Generate(ctx, &ai.Request{
			Model:     c.model,
			System:    "You are terse. Answer in one short sentence.",
			Messages:  []ai.Message{ai.UserText("What is a database index?")},
			MaxTokens: 80,
		})
		if err != nil {
			fmt.Printf("   [%s] error: %v\n", c.name, err)
			continue
		}
		fmt.Printf("   [%s] in=%d out=%d: %s\n", c.name, resp.Usage.InputTokens, resp.Usage.OutputTokens, oneLine(resp.Text()))
	}

	// B. stream: text arrives as it is produced.
	fmt.Printf("B. stream from %s:\n   ", p.name)
	for chunk, err := range p.client.Stream(ctx, &ai.Request{
		Model: p.model, MaxTokens: 60,
		Messages: []ai.Message{ai.UserText("Name three Go standard library packages, comma-separated.")},
	}) {
		if err != nil {
			fmt.Printf("\n   error: %v\n", err)
			break
		}
		fmt.Print(chunk.Text)
		if chunk.Done {
			fmt.Printf("\n   (done, out=%d tokens)\n", chunk.Usage.OutputTokens)
		}
	}

	// C. shape: System + a tiny MaxTokens force one word.
	fmt.Printf("C. shape with System + MaxTokens (%s):\n", p.name)
	resp, err := p.client.Generate(ctx, &ai.Request{
		Model: p.model, System: "Reply with exactly one word and nothing else.",
		Messages: []ai.Message{ai.UserText("The capital of France?")}, MaxTokens: 5,
	})
	if err != nil {
		return err
	}
	fmt.Printf("   -> %q\n", oneLine(resp.Text()))

	// D. photo (vision): send inline image bytes in a message. A message can mix
	// a Text part and an Image part; the Image carries the MIME type and bytes.
	fmt.Printf("D. photo / vision (%s):\n", p.name)
	pngBytes := redSquarePNG()
	resp, err = p.client.Generate(ctx, &ai.Request{
		Model:     p.model,
		MaxTokens: 20,
		Messages: []ai.Message{{
			Role: ai.RoleUser,
			Parts: []ai.Part{
				ai.Text{Text: "What color is this square? Answer with one word."},
				ai.Image{MIME: "image/png", Data: pngBytes},
			},
		}},
	})
	if err != nil {
		return err
	}
	fmt.Printf("   sent a %d-byte PNG, model saw: %q\n", len(pngBytes), oneLine(resp.Text()))

	// E. tools: describe a function, let the model call it, answer the call, and
	// get the final reply. This is how a model reaches out to your code.
	fmt.Printf("E. tools / function calling (%s):\n", p.name)
	weather := ai.Tool{
		Name:        "get_weather",
		Description: "Get the current weather for a city",
		Schema:      json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}`),
	}
	first, err := p.client.Generate(ctx, &ai.Request{
		Model: p.model, MaxTokens: 200,
		Tools: []ai.Tool{weather}, ToolChoice: ai.ToolAuto,
		Messages: []ai.Message{ai.UserText("What is the weather in Kyiv? Use the tool.")},
	})
	if err != nil {
		return err
	}
	calls := first.ToolCalls()
	if len(calls) == 0 {
		fmt.Printf("   model answered without a tool call: %q\n", oneLine(first.Text()))
		return nil
	}
	call := calls[0]
	fmt.Printf("   model called %s(%s)\n", call.Name, call.Input)

	// Answer the tool call and ask again; the model turns the data into prose.
	final, err := p.client.Generate(ctx, &ai.Request{
		Model: p.model, MaxTokens: 80,
		Tools: []ai.Tool{weather},
		Messages: []ai.Message{
			ai.UserText("What is the weather in Kyiv? Use the tool."),
			{Role: ai.RoleAssistant, Parts: []ai.Part{call}},
			{Role: ai.RoleTool, Parts: []ai.Part{ai.ToolResult{ID: call.ID, Content: `{"temp_c":15,"sky":"sunny"}`}}},
		},
	})
	if err != nil {
		return err
	}
	fmt.Printf("   after the tool result: %s\n", oneLine(final.Text()))
	return nil
}

// redSquarePNG builds a small solid-red PNG in memory, so the vision example
// needs no asset file on disk.
func redSquarePNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 48, 48))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{R: 210, G: 30, B: 30, A: 255}}, image.Point{}, draw.Src)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// oneLine collapses newlines so a multi-line answer prints on one report line.
func oneLine(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '\n' || r == '\r' {
			out = append(out, ' ')
			continue
		}
		out = append(out, r)
	}
	return string(out)
}
