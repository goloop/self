package main

import (
	"encoding/json"
	"testing"

	"github.com/goloop/ai"
)

// The schema the feature ships must be strict-compatible, and this test is why
// example D can promise that: a typo in storySchema fails here, at build time,
// instead of as a 400 on a user's key after a deploy.
func TestStorySchemaIsStrictCompatible(t *testing.T) {
	if err := ai.ValidateStrictSchema(json.RawMessage(storySchema)); err != nil {
		t.Fatalf("storySchema is not strict-compatible: %v", err)
	}
}

// A worked counter-example: the same schema missing a property from required
// must be rejected, so the check above is known to have teeth.
func TestStrictSchemaCatchesAMissingRequired(t *testing.T) {
	bad := `{
	  "type": "object",
	  "additionalProperties": false,
	  "required": ["headline"],
	  "properties": {
	    "headline": {"type": "string"},
	    "summary":  {"type": "string"}
	  }}`
	if err := ai.ValidateStrictSchema(json.RawMessage(bad)); err == nil {
		t.Error("a schema missing 'summary' from required was accepted")
	}
}

// The factory maps known names to clients and rejects unknown ones. It needs
// no key: construction does not call the network.
func TestFactory(t *testing.T) {
	for _, name := range []string{"anthropic", "openai"} {
		c, err := newClient(name, "probe")
		if err != nil || c == nil {
			t.Errorf("newClient(%q) = %v, %v", name, c, err)
		}
	}
	if _, err := newClient("nope", "k"); err == nil {
		t.Error("newClient accepted an unknown provider")
	}
}

// Capabilities are static, so this holds with no key and no network - which is
// the whole point of asking the driver instead of maintaining a provider table.
func TestCapabilitiesAreKnownWithoutAKey(t *testing.T) {
	anthropicC, _ := newClient("anthropic", "probe")
	if !ai.SupportsHosted(anthropicC, ai.Hosted{Kind: ai.HostedWebSearch}) {
		t.Error("anthropic should report web search support")
	}

	openaiC, _ := newClient("openai", "probe")
	hc, ok := ai.HostedCapabilityOf(openaiC, ai.HostedWebSearch)
	if !ok {
		t.Fatal("openai should report a web search capability")
	}
	// openai carries the format on the responses endpoint, so a search and a
	// schema fit in one call; the recipe's example B prints exactly this.
	if hc.WithFormat.JSONSchema != ai.FormatNative {
		t.Errorf("openai WithFormat.JSONSchema = %v, want native", hc.WithFormat.JSONSchema)
	}
}

// The recipe runs to completion with no keys set: the live section skips.
func TestRunWithoutKeys(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	if err := run(); err != nil {
		t.Fatal(err)
	}
}
