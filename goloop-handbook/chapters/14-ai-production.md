[« Behind a reverse proxy](13-reverse-proxy.md) · [Contents](../main.md) · [Errors a frontend can switch on »](15-error-slugs.md)

---

# 14. AI, production shape

**Task.** Chapter 05 made the calls; this one is the shape a real service wraps
around them. Draw it from services that let a user bring their own provider key,
ask a model to research something, and want the answer in a schema. Three
problems show up that a single call does not have: which provider to construct
and whether it can even do what you are about to ask; how to degrade when it
cannot, without matching English error prose; and how to catch a schema mistake
before it becomes a 400 on the user's own key after a deploy.

**Modules.** [`ai`](https://github.com/goloop/ai) is the provider-agnostic
contract - `Client`, `Capabilities`, the sentinels and `ValidateStrictSchema` -
and the driver packages
([`anthropic`](https://github.com/goloop/anthropic),
[`openai`](https://github.com/goloop/openai) and the rest) implement it.

**Recipe.** [`recipes/014-ai-production`](../recipes/014-ai-production/)

Like recipe 005, this compiles and tests green with **no API keys**: it builds a
client only for a provider whose key is present, and the live call prints a note
and skips when none is. The capability and schema work needs no network at all.

## Example A - the factory

One function maps a provider name and key to an `ai.Client`, so nothing else in
the code is provider-specific:

```go
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
```

A real service reads the name and key from the account, so each user brings
their own provider - and the rest of the feature never knows which one.

## Example B - ask what it can do, before the call

Whether to show a "search the web" control is a decision you make *before* a
call. The instinct is a table of provider names; the projects this is drawn from
kept that table in three places - backend twice, frontend once - and all three
went out of date the day a driver learned to search. Ask the driver instead:

```go
if ai.SupportsHosted(client, ai.Hosted{Kind: ai.HostedWebSearch}) {
	// offer the control
}
hc, _ := ai.HostedCapabilityOf(client, ai.HostedWebSearch)
// hc.WithFormat.JSONSchema tells you whether search + schema fit in one call
```

`Capabilities` is a **hint, not a permission**: whether a request actually
succeeds still depends on the model, the account and the region, so the
sentinels in example C remain the truth. But it is the right hint for the
before-the-call decision - offer the feature, and one request or two - and a
driver that does not describe itself reports nothing rather than a wrong yes.

## Example C - degrade on the sentinel, not the prose

When a provider cannot do what was asked, the driver says so through its own
sentinel, and the code branches on `errors.Is`:

```go
resp, err := client.Generate(ctx, req) // req asks for a web search
switch {
case err == nil:
	// use resp.Citations()
case errors.Is(err, ai.ErrNoHosted):
	// this provider cannot search: fall back to a plain answer, or ask
	// the user to switch providers - do not answer from memory and pretend
case errors.Is(err, ai.ErrFormatWithHosted):
	// it will not search AND enforce a schema at once: make two calls -
	// search in prose, then reshape into the schema
default:
	return err
}
```

This replaced fragile message-matching that two projects had independently
written - each parsing the English of a provider's 400 to guess what failed. The
driver now wraps that 400 in the sentinel itself, so one `errors.Is` covers a
limitation it knew in advance and one it learned over the wire; the provider's
own `*ai.APIError` is still reachable with `errors.As` when you want the detail.

The two-call shape example C mentions - search in prose, then reshape into the
schema - is the production answer to a provider (like Gemini) that enforces a
schema by constraining output, which does not hold together with grounding.
`hc.WithFormat.JSONSchema` from example B tells you in advance which providers
need it.

## Example D - check the schema before you send it

A strict schema is a literal, fully known before any call. A missing entry in
`required` should not be something the provider tells you over the network, on a
user's key, after a deploy. `ai.ValidateStrictSchema` checks the strict-mode
rules - every object needs `additionalProperties: false` and every property in
`required` - and names the path to the fault:

```go
if err := ai.ValidateStrictSchema(storySchema); err != nil {
	// ai: strict schema is not provider-compatible:
	//   properties.stories.items: "required" is missing "placeLocal"
}
```

Put it in a test (the recipe does), and a schema typo is a build failure, not a
production incident. It checks the rules that decide whether strict mode is
accepted, and nothing about whether the schema describes what you meant - that
is still yours.

## Execution report

```
$ go run .                        # with no keys set
A/B. what can each provider be asked for (a hint, before any call):
   anthropic web search=true   search+schema in one call=emulated
   openai    web search=true   search+schema in one call=native
   -> the UI shows the search control only where it is supported
D. strict schema checked locally before any call:
   storySchema is strict-compatible
C. live call: skipped - set ANTHROPIC_API_KEY or OPENAI_API_KEY
   (compile and tests are green without any key)
```

Note the capability table above: `anthropic` reports `emulated` for search +
schema (its structured output is prompt-emulated, so the two coexist but nothing
is enforced), `openai` reports `native` (its responses endpoint carries the
format, so search and a strict schema fit in one call). That is exactly the
knowledge the projects were hand-maintaining, now read from the driver.

## What you learned

- A factory turns a provider name + key into an `ai.Client`; keep the rest
  provider-agnostic.
- `ai.SupportsHosted` / `ai.HostedCapabilityOf` answer before the call - whether
  to offer a feature and whether it takes one request or two - replacing a
  provider-name table that rots. It is a hint, not a permission.
- Degrade on the driver's own sentinels (`ai.ErrNoHosted`, `ai.ErrNoFormat`,
  `ai.ErrFormatWithHosted`) with `errors.Is`, never on the text of a 400; the
  `*ai.APIError` stays reachable with `errors.As`.
- `ai.ValidateStrictSchema` in a test turns a schema mistake into a build
  failure instead of a live 400 on a user's key.

Next: the errors this all produces, shaped so a frontend can act on them.

---

[« Behind a reverse proxy](13-reverse-proxy.md) · [Contents](../main.md) · [Errors a frontend can switch on »](15-error-slugs.md)
