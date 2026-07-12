# Recipe: 005-ai

Part of the [GoLoop One handbook](../../main.md). Read the chapter:
[05-ai](../../chapters/05-ai.md).

Run it against the real APIs:

```sh
export ANTHROPIC_API_KEY=sk-ant-...   # set either or both
export OPENAI_API_KEY=sk-proj-...

go test ./...     # one cheap call per configured provider; skips with no key
go run .
```

The program uses `goloop/ai` as the interface and `anthropic` / `openai` as the
drivers. A provider is used only when its key is set, so it runs with one or
both. No API key is stored in this repository.
