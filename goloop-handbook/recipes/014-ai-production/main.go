// Recipe 014: an AI feature shaped for production.
//
// Recipe 005 showed the calls. This one shows the shape a real service wraps
// around them, drawn from projects that let a user bring their own provider key
// and ask a model to research something and answer in a schema:
//
//	A. a factory      - one function turns a provider name and key into an
//	                    ai.Client, so the rest of the code is provider-agnostic;
//	B. capabilities   - ai.SupportsHosted answers, before the call, whether to
//	                    offer a "search the web" control at all - instead of a
//	                    hand-written table of provider names that rots;
//	C. honest degrade - when a provider cannot do what was asked, the driver's
//	                    own sentinel (ai.ErrNoHosted / ai.ErrNoFormat /
//	                    ai.ErrFormatWithHosted) says so, and the code branches
//	                    on errors.Is rather than matching English error prose;
//	D. schema safety  - a strict schema is checked with ai.ValidateStrictSchema
//	                    before it is ever sent, so a typo is a build-time test
//	                    failure and not a 400 on the user's key after a deploy.
//
// Like recipe 005, this runs and tests green with NO keys: it constructs a
// client only for a provider whose key is present, and the live part prints a
// note and skips when none is. Set ANTHROPIC_API_KEY or OPENAI_API_KEY to see
// the live call. No key is stored in this repository.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/goloop/ai"
	"github.com/goloop/anthropic"
	"github.com/goloop/openai"
)

// provider is one configured backend: a display name, a model, and the client.
type provider struct {
	name   string
	model  string
	client ai.Client
}

// newClient is the factory. It maps a provider name to a driver; every driver
// returns an ai.Client, so nothing downstream is provider-specific. A real
// service reads the name and key from the account, not from the environment.
func newClient(name, key string) (ai.Client, error) {
	switch name {
	case "anthropic":
		return anthropic.New(key), nil
	case "openai":
		return openai.New(key), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", name)
	}
}

// configured returns the providers whose keys are set, in a stable order. With
// no keys it returns nothing, and the program still runs - the capability and
// schema checks below need no network.
func configured() []provider {
	var out []provider
	if k := os.Getenv("ANTHROPIC_API_KEY"); k != "" {
		c, _ := newClient("anthropic", k)
		out = append(out, provider{"anthropic", anthropic.ModelClaudeHaiku45, c})
	}
	if k := os.Getenv("OPENAI_API_KEY"); k != "" {
		c, _ := newClient("openai", k)
		out = append(out, provider{"openai", openai.ModelGPT4oMini, c})
	}
	return out
}

// storySchema is the strict schema the feature wants its answer in. It is a
// literal, fully known before any call - which is exactly why example D checks
// it at build time.
const storySchema = `{
  "type": "object",
  "additionalProperties": false,
  "required": ["headline", "summary"],
  "properties": {
    "headline": {"type": "string"},
    "summary":  {"type": "string"}
  }
}`

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "recipe:", err)
		os.Exit(1)
	}
}

func run() error {
	// A. + B. Capability checks, no network required. The map of provider
	// names to "can it search" lived in three places in the projects this is
	// drawn from - backend twice and the frontend once - and went out of date
	// the day a driver learned to search. Asking the driver keeps one answer.
	fmt.Println("A/B. what can each provider be asked for (a hint, before any call):")
	for _, name := range []string{"anthropic", "openai"} {
		// A probe client needs no real key: capabilities are static.
		c, _ := newClient(name, "probe")
		search := ai.SupportsHosted(c, ai.Hosted{Kind: ai.HostedWebSearch})
		hc, _ := ai.HostedCapabilityOf(c, ai.HostedWebSearch)
		fmt.Printf("   %-9s web search=%-5v  search+schema in one call=%s\n",
			name, search, hc.WithFormat.JSONSchema)
	}
	fmt.Println("   -> the UI shows the search control only where it is supported")

	// D. Validate the strict schema before sending it. This is a build-time
	// check: the recipe's test calls ValidateStrictSchema on storySchema, so
	// a mistake in it fails `go test`, not a live request.
	fmt.Println("D. strict schema checked locally before any call:")
	if err := ai.ValidateStrictSchema(json.RawMessage(storySchema)); err != nil {
		return fmt.Errorf("schema is not strict-compatible: %w", err)
	}
	fmt.Println("   storySchema is strict-compatible")

	// C. The live call, only where a key is set. Everything above ran with
	// none.
	providers := configured()
	if len(providers) == 0 {
		fmt.Println("C. live call: skipped - set ANTHROPIC_API_KEY or OPENAI_API_KEY")
		fmt.Println("   (compile and tests are green without any key)")
		return nil
	}

	fmt.Println("C. asking each configured provider, degrading honestly:")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for _, p := range providers {
		if err := ask(ctx, p); err != nil {
			fmt.Printf("   %-9s error: %v\n", p.name, err)
		}
	}
	return nil
}

// ask runs the feature against one provider and reports how it degraded. It
// asks for a web search; a provider that cannot search says so through
// ai.ErrNoHosted, and the code reads that sentinel rather than the prose of a
// 400. That replaced the fragile message-matching two projects had written.
func ask(ctx context.Context, p provider) error {
	req := &ai.Request{
		Model:    p.model,
		Messages: []ai.Message{ai.UserText("In one sentence, what is goloop?")},
		Hosted:   []ai.Hosted{{Kind: ai.HostedWebSearch}},
	}

	resp, err := p.client.Generate(ctx, req)
	switch {
	case err == nil:
		fmt.Printf("   %-9s answered (%d citations)\n", p.name, len(resp.Citations()))
		return nil
	case errors.Is(err, ai.ErrNoHosted):
		// The provider cannot search at all, or not under these constraints.
		// Fall back to a plain answer rather than pretending, or tell the
		// user to switch providers.
		fmt.Printf("   %-9s cannot search; would fall back to a plain answer\n", p.name)
		return nil
	case errors.Is(err, ai.ErrFormatWithHosted):
		// The provider will not combine a search with a strict schema in one
		// call. The feature makes two calls instead: search in prose, then
		// reshape. See the chapter.
		fmt.Printf("   %-9s cannot search AND enforce a schema at once; would use two calls\n", p.name)
		return nil
	default:
		return err
	}
}
