# Recipe 014: an AI feature shaped for production

Part of the [GoLoop One handbook](../../main.md). Read the chapter:
[14. AI, production shape](../../chapters/14-ai-production.md).

```sh
go test ./...   # green with no keys; the strict-schema check has teeth
go run .        # capability table + local schema check; live call skips without a key
```

Set a key to see the live call:

```sh
export ANTHROPIC_API_KEY=sk-ant-...   # either or both
export OPENAI_API_KEY=sk-proj-...
go run .
```

No key is stored in this repository. With none set, the program still runs the
whole capability and schema story - those need no network.

## Environment

| Variable | Meaning |
|---|---|
| `ANTHROPIC_API_KEY` | enables the Anthropic live call (bring your own) |
| `OPENAI_API_KEY` | enables the OpenAI live call (bring your own) |

Copy `.env.example` to `.env`, or export the keys directly.

## The four moves

- **Factory**: one function, provider name + key -> `ai.Client`; everything else
  is provider-agnostic.
- **Capabilities before the call**: `ai.SupportsHosted` decides whether to offer
  a feature, replacing a hand-written provider table that goes stale.
- **Honest degradation**: branch on the driver's own `ai.ErrNoHosted` /
  `ai.ErrNoFormat` / `ai.ErrFormatWithHosted`, never on the text of a 400.
- **Schema safety**: `ai.ValidateStrictSchema` in a test turns a schema typo into
  a build failure instead of a live 400 on a user's key.
